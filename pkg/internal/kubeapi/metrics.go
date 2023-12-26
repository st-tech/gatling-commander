/*
Copyright &copy; ZOZO, Inc.

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the “Software”), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included
in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED “AS IS”, WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

package kubeapi

import (
	"context"
	"fmt"
	"math"
	"os"
	"sync"
	"time"

	"gopkg.in/inf.v0"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	metricsClientset "k8s.io/metrics/pkg/client/clientset/versioned"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"

	cfg "github.com/st-tech/gatling-commander/pkg/config"
)

type metricsPool struct {
	pool []MetricsField
}

func newMetricsPool() *metricsPool {
	return &metricsPool{}
}

// calcEachMetricsMean calcurate mean of each MetricsField value.
func (metricsPool *metricsPool) calcEachMetricsMean() (*MetricsField, error) {
	// fieldExtractor extract field value by given argument function returns value.
	mean := func(fieldExtractor func(metrics MetricsField) int64) (int64, error) {
		if len(metricsPool.pool) <= 0 {
			return 0, fmt.Errorf("input value length is 0")
		}
		sum := int64(0)
		for _, metrics := range metricsPool.pool {
			sum += fieldExtractor(metrics)
		}
		return sum / int64(len(metricsPool.pool)), nil
	}

	cpu, err := mean(func(metrics MetricsField) int64 {
		return metrics.Cpu
	})
	if err != nil {
		return nil, fmt.Errorf("calculate cpu mean error %v", err)
	}

	memory, err := mean(func(metrics MetricsField) int64 {
		return metrics.Memory
	})
	if err != nil {
		return nil, fmt.Errorf("calculate memory mean error %v", err)
	}

	return &MetricsField{
		Cpu:    cpu,
		Memory: memory,
	}, nil
}

func (metricsPool *metricsPool) append(metrics MetricsField) {
	metricsPool.pool = append(metricsPool.pool, metrics)
}

// FetchContainerResourcesLimit fetch specified container and get resources limits or requests field value.
func FetchContainerResourcesLimit(
	ctx context.Context,
	cl ctrlClient.Client,
	podConfig cfg.TargetPodConfig,
) (*MetricsField, error) {
	namespace := podConfig.Namespace
	labelKey := podConfig.LabelKey
	labelValue := podConfig.LabelValue
	targetContainerName := podConfig.ContainerName

	var foundPods corev1.PodList
	labelSelector := labels.SelectorFromSet(labels.Set{labelKey: labelValue})
	listOptions := &ctrlClient.ListOptions{
		Namespace:     namespace,
		LabelSelector: labelSelector,
	}

	err := cl.List(ctx, &foundPods, listOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to list target pod, %v", err)
	}

	if len(foundPods.Items) == 0 {
		return nil, fmt.Errorf("no match pods to specified label")
	}

	var representPod *corev1.Pod
	for _, pod := range foundPods.Items {
		if pod.Status.Phase == "Running" {
			/*
				pod resources value is assumed to be equal among pods which has same label.
				so use one of these as represent.
			*/
			representPod = &pod
			break
		}
	}
	if representPod == nil {
		return nil, fmt.Errorf("founds pods status is not Running")
	}

	for _, c := range representPod.Spec.Containers {
		if c.Name == targetContainerName {
			return getContainerResourcesLimits(c.Resources)
		}
	}
	return nil, fmt.Errorf("target container not found")
}

/*
FetchContainerMetricsMean returns container resources value mean.

Fetch metrics value every 5 seconds until informerCh get value or context done.
Cpu and Memory value is rounded and cast from *inf.Dec to int64.
If error occured this error only log error and continue to run. (not returns error object)
*/
func FetchContainerMetricsMean(
	ctx context.Context,
	wg *sync.WaitGroup,
	cl metricsClientset.Interface,
	resultCh chan MetricsField,
	receiveGatlingFinishedCh chan bool,
	podConfig cfg.TargetPodConfig,
) {
	defer wg.Done()

	namespace := podConfig.Namespace
	labelKey := podConfig.LabelKey
	labelValue := podConfig.LabelValue
	targetContainerName := podConfig.ContainerName

	log := func(msg string, isErr bool) {
		logCommonPodInfo := fmt.Sprintf(
			"Namespace: %v, Label: %v",
			namespace,
			fmt.Sprintf("%v=%v", labelKey, labelValue),
		)
		if isErr {
			fmt.Fprintf(os.Stderr, "Error: %v, %v\n", msg, logCommonPodInfo)
		} else {
			fmt.Printf("%v, %v\n", msg, logCommonPodInfo)
		}
	}

	metricsPool := newMetricsPool()
	for {
		select {
		case <-ctx.Done():
			return
		case <-receiveGatlingFinishedCh:
			log("receive gatling job finished", false)
			meanMetrics, err := metricsPool.calcEachMetricsMean()
			if err != nil {
				log(err.Error(), true)
				return
			}
			resultCh <- *meanMetrics
			return
		default:
			listOptions := metav1.ListOptions{
				LabelSelector: fmt.Sprintf("%v=%v", labelKey, labelValue),
			}
			metricses, err := cl.MetricsV1beta1().PodMetricses(namespace).List(ctx, listOptions)
			if err != nil {
				log(fmt.Sprintf("failed to get pod metricses list %v", err), true)
				continue
			}
			for _, podMetrics := range metricses.Items {
				for _, c := range podMetrics.Containers {
					if c.Name == targetContainerName {
						// The unit of CPU value that the container object has is cores.
						// ref: https://github.com/kubernetes/api/blob/5e9982075c8d828d9501e306ed0a2e133f1ebd88/core/v1/types.go#L5704
						// CPU, in cores. (500 mCPU = .5 vCPU)
						cpuUsageDec := c.Usage.Cpu().AsDec()
						// change unit to mCPU and convert *inf.Dec to int64
						// The Round method returns a 3-digit reounded up value which has type inf.Dec.
						// The inf.Dec object Unscaled method returns unscaled property value which has int64 type.
						// nolint:lll // ex: cpuUsageDec: inf.Dec{unscale: 5005, scale: 4} (equal to float64 0.5005), rounded up with scale 3 result: inf.Dec{unscale: 501, scale: 3}, Unscaled result: int64 501
						// So if get 0.5005 vCPU it convert 501 mCPU
						cpuUsage, ok := cpuUsageDec.Round(cpuUsageDec, 3, inf.RoundUp).Unscaled()
						if !ok {
							log("failed to convert value of container metrics cpu usage *inf.Dec to int64", true)
						}
						// The unit of Memory value that the container object has is bytes.
						// ref: https://github.com/kubernetes/api/blob/5e9982075c8d828d9501e306ed0a2e133f1ebd88/core/v1/types.go#L5704
						memUsage, _ := c.Usage.Memory().AsInt64()
						metricsPool.append(
							MetricsField{
								Cpu:    cpuUsage,
								Memory: memUsage,
							},
						)
					}
				}
			}
			time.Sleep(5 * time.Second) // NOTE: wait few seconds to avoid excessive cpu usage.
		}
	}
}

// CalcAndRoundMetricsRatio calculate ratio by resource usage and limit.
func CalcAndRoundMetricsRatio(usage, limit int64) float64 {
	ratio := float64(usage) / float64(limit)
	roundedRatio := math.Round(ratio*1000) / 1000 // round 4th place to closest
	return roundedRatio
}

/*
getContainerResourcesLimits returns container resources limits field value.
If the limits field is not present, get requests field value.
If both of these are not present, an error is returned
*/
func getContainerResourcesLimits(resources corev1.ResourceRequirements) (*MetricsField, error) {
	var (
		gotCpu *inf.Dec
		gotMem int64
	)
	// if Limits field value is not set, Quantity AsDec() method return inf.Dec{unscale: 0, scale: 0}
	// nolint:lll // ref: https://github.com/kubernetes/apimachinery/blob/fd8daa85285e31da9771dbe372a66dfa20e78489/pkg/api/resource/quantity.go#L500
	// nolint:lll // ref: https://github.com/kubernetes/apimachinery/blob/fd8daa85285e31da9771dbe372a66dfa20e78489/pkg/api/resource/amount.go#L104
	if lim := resources.Limits.Cpu().AsDec(); lim.Cmp(inf.NewDec(0, 0)) == 0 {
		req := resources.Requests.Cpu().AsDec()
		if req.Cmp(inf.NewDec(0, 0)) == 0 {
			return nil, fmt.Errorf("failed to get container metrics cpu both limits and requests")
		}
		gotCpu = req // use requests value as limits value if limits field value is not set.
	} else {
		gotCpu = lim
	}

	if lim, _ := resources.Limits.Memory().AsInt64(); lim == 0 {
		req, _ := resources.Requests.Memory().AsInt64()
		if req == 0 {
			return nil, fmt.Errorf("failed to get container metrics memory both limits and requests")
		}
		gotMem = req // use requests value as limits value if limits field value is not set.
	} else {
		gotMem = lim
	}

	/*
		got cpu unit is cores.
		ref: https://github.com/kubernetes/api/blob/5e9982075c8d828d9501e306ed0a2e133f1ebd88/core/v1/types.go#L5704
		CPU, in cores. (500m = .5 cores)
		change unit to m vCPU and convert *inf.Dec to int64
	*/
	cpuLimits, ok := gotCpu.Round(gotCpu, 3, inf.RoundUp).Unscaled()
	if !ok {
		return nil, fmt.Errorf("failed to round and unscaled cpu value")
	}

	/*
		got memory unit is bytes.
		ref: https://github.com/kubernetes/api/blob/5e9982075c8d828d9501e306ed0a2e133f1ebd88/core/v1/types.go#L5704
	*/
	memLimits := gotMem

	return &MetricsField{
		Cpu:    cpuLimits,
		Memory: memLimits,
	}, nil
}
