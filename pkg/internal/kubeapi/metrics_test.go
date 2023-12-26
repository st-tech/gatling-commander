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
	"sync"
	"testing"
	"time"

	cfg "github.com/st-tech/gatling-commander/pkg/config"
	"github.com/st-tech/gatling-commander/pkg/internal/kubeutil"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/metrics/pkg/apis/metrics/v1beta1"
	metricsFake "k8s.io/metrics/pkg/client/clientset/versioned/fake"
)

func TestFetchContainerResourcesLimit(t *testing.T) {
	cl := kubeutil.InitFakeClient()
	const (
		labelKey = "app"
		podCpu   = int64(500)                     // m vCPU
		podMem   = int64(10 * 1024 * 1024 * 1024) // bytes
	)

	samplePod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sample-pod",
			Namespace: "sample-namespace",
			Labels: map[string]string{
				labelKey: "sample-app",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				corev1.Container{
					Name: "sample-container",
					Resources: corev1.ResourceRequirements{
						Limits: v1.ResourceList{
							v1.ResourceCPU:    resource.MustParse(fmt.Sprintf("%vm", podCpu)),
							v1.ResourceMemory: resource.MustParse(fmt.Sprintf("%v", podMem)),
						},
					},
				},
			},
		},
		Status: corev1.PodStatus{
			Phase: "Running",
		},
	}

	samplePodOnlyResourcesRequests := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sample-pod-only-resources-requests",
			Namespace: "sample-namespace",
			Labels: map[string]string{
				labelKey: "sample-app-only-resources-requests",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				corev1.Container{
					Name: "sample-container-only-resources-requests",
					Resources: corev1.ResourceRequirements{
						Requests: v1.ResourceList{
							v1.ResourceCPU:    resource.MustParse(fmt.Sprintf("%vm", podCpu)),
							v1.ResourceMemory: resource.MustParse(fmt.Sprintf("%v", podMem)),
						},
					},
				},
			},
		},
		Status: corev1.PodStatus{
			Phase: "Running",
		},
	}

	err := cl.Create(context.TODO(), &samplePod)
	assert.NoError(t, err)
	err = cl.Create(context.TODO(), &samplePodOnlyResourcesRequests)
	assert.NoError(t, err)

	samplePodConfig := cfg.TargetPodConfig{
		Namespace:     samplePod.ObjectMeta.Namespace,
		LabelKey:      labelKey,
		LabelValue:    samplePod.ObjectMeta.Labels[labelKey],
		ContainerName: samplePod.Spec.Containers[0].Name,
	}
	samplePodOnlyResourcesRequestsConfig := cfg.TargetPodConfig{
		Namespace:     samplePodOnlyResourcesRequests.ObjectMeta.Namespace,
		LabelKey:      labelKey,
		LabelValue:    samplePodOnlyResourcesRequests.ObjectMeta.Labels[labelKey],
		ContainerName: samplePodOnlyResourcesRequests.Spec.Containers[0].Name,
	}

	expectedResourcesLimit := &MetricsField{
		Cpu:    podCpu,
		Memory: podMem,
	}

	tests := []struct {
		name      string
		podConfig cfg.TargetPodConfig
	}{
		{
			name:      "success to fetched resources limits value",
			podConfig: samplePodConfig,
		},
		{
			name:      "success to fetched resources requests value",
			podConfig: samplePodOnlyResourcesRequestsConfig,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			containerResourcesLimit, err := FetchContainerResourcesLimit(context.TODO(), cl, tt.podConfig)
			assert.NoError(t, err)
			assert.Equal(t, expectedResourcesLimit, containerResourcesLimit)
		})
	}
}

func TestFetchContainerResourcesLimit_MultiMemUnit(t *testing.T) {
	cl := kubeutil.InitFakeClient()

	const (
		podCpu   = int64(500)                           // m vCPU
		podMem   = int64(10 * 1024 * 1024 * 1024)       // bytes sample value 10Gi
		podMemMi = int64(podMem / (1024 * 1024))        // Mi
		podMemGi = int64(podMem / (1024 * 1024 * 1024)) // Gi
		labelKey = "app"
	)

	samplePodMemBytes := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sample-pod-mem-bytes",
			Namespace: "sample-namespace",
			Labels: map[string]string{
				labelKey: "sample-app-mem-bytes",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				corev1.Container{
					Name: "sample-container-mem-bytes",
					Resources: corev1.ResourceRequirements{
						Limits: v1.ResourceList{
							v1.ResourceCPU:    resource.MustParse(fmt.Sprintf("%vm", podCpu)),
							v1.ResourceMemory: resource.MustParse(fmt.Sprintf("%v", podMem)),
						},
					},
				},
			},
		},
		Status: corev1.PodStatus{
			Phase: "Running",
		},
	}

	samplePodMemMi := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sample-pod-mem-mi",
			Namespace: "sample-namespace",
			Labels: map[string]string{
				labelKey: "sample-app-mem-mi",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				corev1.Container{
					Name: "sample-container-mem-mi",
					Resources: corev1.ResourceRequirements{
						Limits: v1.ResourceList{
							v1.ResourceCPU:    resource.MustParse(fmt.Sprintf("%vm", podCpu)),
							v1.ResourceMemory: resource.MustParse(fmt.Sprintf("%vMi", podMemMi)),
						},
					},
				},
			},
		},
		Status: corev1.PodStatus{
			Phase: "Running",
		},
	}

	samplePodMemGi := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sample-pod-mem-gi",
			Namespace: "sample-namespace",
			Labels: map[string]string{
				labelKey: "sample-app-mem-gi",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				corev1.Container{
					Name: "sample-container-mem-gi",
					Resources: corev1.ResourceRequirements{
						Limits: v1.ResourceList{
							v1.ResourceCPU:    resource.MustParse(fmt.Sprintf("%vm", podCpu)),
							v1.ResourceMemory: resource.MustParse(fmt.Sprintf("%vGi", podMemGi)),
						},
					},
				},
			},
		},
		Status: corev1.PodStatus{
			Phase: "Running",
		},
	}

	err := cl.Create(context.TODO(), &samplePodMemBytes)
	assert.NoError(t, err)
	err = cl.Create(context.TODO(), &samplePodMemMi)
	assert.NoError(t, err)
	err = cl.Create(context.TODO(), &samplePodMemGi)
	assert.NoError(t, err)

	memBytesPodConfig := cfg.TargetPodConfig{
		Namespace:     samplePodMemBytes.ObjectMeta.Namespace,
		LabelKey:      labelKey,
		LabelValue:    samplePodMemBytes.ObjectMeta.Labels[labelKey],
		ContainerName: samplePodMemBytes.Spec.Containers[0].Name,
	}
	memMiPodConfig := cfg.TargetPodConfig{
		Namespace:     samplePodMemMi.ObjectMeta.Namespace,
		LabelKey:      labelKey,
		LabelValue:    samplePodMemMi.ObjectMeta.Labels[labelKey],
		ContainerName: samplePodMemMi.Spec.Containers[0].Name,
	}
	memGiPodConfig := cfg.TargetPodConfig{
		Namespace:     samplePodMemGi.ObjectMeta.Namespace,
		LabelKey:      labelKey,
		LabelValue:    samplePodMemGi.ObjectMeta.Labels[labelKey],
		ContainerName: samplePodMemGi.Spec.Containers[0].Name,
	}

	expectedResourcesLimit := &MetricsField{
		Cpu:    podCpu,
		Memory: podMem,
	}

	tests := []struct {
		name      string
		podConfig cfg.TargetPodConfig
	}{
		{
			name:      "memory limit bytes",
			podConfig: memBytesPodConfig,
		},
		{
			name:      "memory limit Mi",
			podConfig: memMiPodConfig,
		},
		{
			name:      "memory limit Gi",
			podConfig: memGiPodConfig,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			containerResourcesLimit, err := FetchContainerResourcesLimit(context.TODO(), cl, tt.podConfig)
			assert.NoError(t, err)
			assert.Equal(t, expectedResourcesLimit, containerResourcesLimit)
		})
	}
}

func TestFetchContainerResourcesLimit_Failed(t *testing.T) {
	cl := kubeutil.InitFakeClient()
	const (
		podCpu = int64(500)                     // m vCPU
		podMem = int64(10 * 1024 * 1024 * 1024) // bytes sample value 10Gi
	)

	labelKey := "app"
	samplePod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sample-pod",
			Namespace: "sample-namespace",
			Labels: map[string]string{
				labelKey: "sample-app",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				corev1.Container{
					Name: "sample-container",
					Resources: corev1.ResourceRequirements{
						Limits: v1.ResourceList{
							v1.ResourceCPU:    resource.MustParse(fmt.Sprintf("%vm", podCpu)),
							v1.ResourceMemory: resource.MustParse(fmt.Sprintf("%v", podMem)),
						},
					},
				},
			},
		},
		Status: corev1.PodStatus{
			Phase: "Running",
		},
	}
	statusNotRunningPod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "status-not-running-sample-pod",
			Namespace: "sample-namespace",
			Labels: map[string]string{
				labelKey: "status-not-running-sample-app",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				corev1.Container{
					Name: "status-not-running-sample-container",
					Resources: corev1.ResourceRequirements{
						Limits: v1.ResourceList{
							v1.ResourceCPU:    resource.MustParse(fmt.Sprintf("%vm", podCpu)),
							v1.ResourceMemory: resource.MustParse(fmt.Sprintf("%vGi", podMem)),
						},
					},
				},
			},
		},
		Status: corev1.PodStatus{
			Phase: "Pending",
		},
	}
	noResourcesMemFieldPod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "no-resources-mem-field-sample-pod",
			Namespace: "sample-namespace",
			Labels: map[string]string{
				labelKey: "no-resources-mem-field-sample-app",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				corev1.Container{
					Name: "no-resources-mem-field-sample-container",
					Resources: corev1.ResourceRequirements{
						Limits: v1.ResourceList{
							v1.ResourceCPU: resource.MustParse(fmt.Sprintf("%vm", podCpu)),
						},
					},
				},
			},
		},
		Status: corev1.PodStatus{
			Phase: "Running",
		},
	}
	noResourcesCpuFieldPod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "no-resources-cpu-field-sample-pod",
			Namespace: "sample-namespace",
			Labels: map[string]string{
				labelKey: "no-resources-cpu-field-sample-app",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				corev1.Container{
					Name: "no-resources-cpu-field-sample-container",
					Resources: corev1.ResourceRequirements{
						Limits: v1.ResourceList{
							v1.ResourceMemory: resource.MustParse(fmt.Sprintf("%vGi", podMem)),
						},
					},
				},
			},
		},
		Status: corev1.PodStatus{
			Phase: "Running",
		},
	}

	err := cl.Create(context.TODO(), &samplePod)
	assert.NoError(t, err)
	err = cl.Create(context.TODO(), &statusNotRunningPod)
	assert.NoError(t, err)
	err = cl.Create(context.TODO(), &noResourcesMemFieldPod)
	assert.NoError(t, err)
	err = cl.Create(context.TODO(), &noResourcesCpuFieldPod)
	assert.NoError(t, err)

	notExistsPodConfig := cfg.TargetPodConfig{
		Namespace:     "not exists namespace",
		LabelKey:      labelKey,
		LabelValue:    samplePod.ObjectMeta.Labels[labelKey],
		ContainerName: samplePod.Spec.Containers[0].Name,
	}
	statusNotRunningPodConfig := cfg.TargetPodConfig{
		Namespace:     statusNotRunningPod.ObjectMeta.Namespace,
		LabelKey:      labelKey,
		LabelValue:    statusNotRunningPod.ObjectMeta.Labels[labelKey],
		ContainerName: statusNotRunningPod.Spec.Containers[0].Name,
	}
	noResourcesMemFieldPodConfig := cfg.TargetPodConfig{
		Namespace:     noResourcesMemFieldPod.ObjectMeta.Namespace,
		LabelKey:      labelKey,
		LabelValue:    noResourcesMemFieldPod.ObjectMeta.Labels[labelKey],
		ContainerName: noResourcesMemFieldPod.Spec.Containers[0].Name,
	}
	noResourcesCpuFieldPodConfig := cfg.TargetPodConfig{
		Namespace:     noResourcesCpuFieldPod.ObjectMeta.Namespace,
		LabelKey:      labelKey,
		LabelValue:    noResourcesCpuFieldPod.ObjectMeta.Labels[labelKey],
		ContainerName: noResourcesCpuFieldPod.Spec.Containers[0].Name,
	}

	tests := []struct {
		name      string
		podConfig cfg.TargetPodConfig
		expected  error
	}{
		{
			name:      "failed because target pod not exists",
			podConfig: notExistsPodConfig,
			expected:  fmt.Errorf("no match pods to specified label"),
		},
		{
			name:      "failed because target pod with status Running is not exists",
			podConfig: statusNotRunningPodConfig,
			expected:  fmt.Errorf("founds pods status is not Running"),
		},
		{
			name:      "container has no resources memory field",
			podConfig: noResourcesMemFieldPodConfig,
			expected:  fmt.Errorf("failed to get container metrics memory both limits and requests"),
		},
		{
			name:      "container has no resources cpu field",
			podConfig: noResourcesCpuFieldPodConfig,
			expected:  fmt.Errorf("failed to get container metrics cpu both limits and requests"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := FetchContainerResourcesLimit(context.TODO(), cl, tt.podConfig)
			assert.Equal(t, tt.expected, err)
		})
	}
}

func TestGetContainerResourcesLimits(t *testing.T) {
	const (
		podCpu = int64(500)                     // m vCPU
		podMem = int64(10 * 1024 * 1024 * 1024) // bytes sample value 10Gi
	)

	tests := []struct {
		name  string
		input corev1.ResourceRequirements
	}{
		{
			name: "success to get cpu requests value",
			input: corev1.ResourceRequirements{
				Limits: v1.ResourceList{
					v1.ResourceMemory: resource.MustParse(fmt.Sprintf("%v", podMem)),
				},
				Requests: v1.ResourceList{
					v1.ResourceCPU: resource.MustParse(fmt.Sprintf("%vm", podCpu)),
				},
			},
		},
		{
			name: "success to get cpu limits value",
			input: corev1.ResourceRequirements{
				Limits: v1.ResourceList{
					v1.ResourceCPU:    resource.MustParse(fmt.Sprintf("%vm", podCpu)),
					v1.ResourceMemory: resource.MustParse(fmt.Sprintf("%v", podMem)),
				},
			},
		},
		{
			name: "success to get memory requests value",
			input: corev1.ResourceRequirements{
				Limits: v1.ResourceList{
					v1.ResourceCPU: resource.MustParse(fmt.Sprintf("%vm", podCpu)),
				},
				Requests: v1.ResourceList{
					v1.ResourceMemory: resource.MustParse(fmt.Sprintf("%v", podMem)),
				},
			},
		},
		{
			name: "success to get memory limits value",
			input: corev1.ResourceRequirements{
				Limits: v1.ResourceList{
					v1.ResourceCPU:    resource.MustParse(fmt.Sprintf("%vm", podCpu)),
					v1.ResourceMemory: resource.MustParse(fmt.Sprintf("%v", podMem)),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			containerResourcesLimit, err := getContainerResourcesLimits(tt.input)
			assert.NoError(t, err)
			assert.Equal(t, &MetricsField{Cpu: podCpu, Memory: podMem}, containerResourcesLimit)
		})
	}
}

func TestGetContainerResourcesLimits_Failed(t *testing.T) {
	const (
		podCpu = int64(500)                     // m vCPU
		podMem = int64(10 * 1024 * 1024 * 1024) // bytes sample value 10Gi
	)

	tests := []struct {
		name     string
		input    corev1.ResourceRequirements
		expected error
	}{
		{
			name: "failed to get cpu value",
			input: corev1.ResourceRequirements{
				Limits: v1.ResourceList{
					v1.ResourceMemory: resource.MustParse(fmt.Sprintf("%v", podMem)),
				},
			},
			expected: fmt.Errorf("failed to get container metrics cpu both limits and requests"),
		},
		{
			name: "failed to get memory value",
			input: corev1.ResourceRequirements{
				Limits: v1.ResourceList{
					v1.ResourceCPU: resource.MustParse(fmt.Sprintf("%vm", podCpu)),
				},
			},
			expected: fmt.Errorf("failed to get container metrics memory both limits and requests"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := getContainerResourcesLimits(tt.input)
			assert.Equal(t, tt.expected, err)
		})
	}
}

func TestFetchTargetPodMetricsMean(t *testing.T) {
	ctx := context.TODO()

	const (
		podCpu = int64(500)                     // m vCPU
		podMem = int64(10 * 1024 * 1024 * 1024) // bytes
	)
	podLabel := map[string]string{
		"app": "sample-app",
	}
	podMetrics := &v1beta1.PodMetrics{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sample-pod",
			Namespace: "sample-namespace",
			Labels:    podLabel,
		},
		Containers: []v1beta1.ContainerMetrics{
			{
				Name: "sample-container",
				Usage: v1.ResourceList{
					v1.ResourceCPU:    resource.MustParse(fmt.Sprintf("%vm", podCpu)),
					v1.ResourceMemory: resource.MustParse(fmt.Sprintf("%v", podMem)),
				},
			},
		},
	}

	gvr := schema.GroupVersionResource{Group: "metrics.k8s.io", Version: "v1beta1", Resource: "pods"}
	cl := metricsFake.NewSimpleClientset(podMetrics)
	_ = cl.Tracker().Create(gvr, podMetrics, podMetrics.ObjectMeta.Namespace)

	var targetLabelKey, targetLabelVal string
	for key, val := range podMetrics.ObjectMeta.Labels {
		targetLabelKey = key
		targetLabelVal = val
	}

	podConfig := cfg.TargetPodConfig{
		Namespace:     podMetrics.ObjectMeta.Namespace,
		LabelKey:      targetLabelKey,
		LabelValue:    targetLabelVal,
		ContainerName: podMetrics.Containers[0].Name,
	}

	tests := []struct {
		name      string
		podConfig cfg.TargetPodConfig
		expected  MetricsField
	}{
		{
			name:      "success to fetch target pod metrics",
			podConfig: podConfig,
			expected: MetricsField{
				Cpu:    podCpu,
				Memory: podMem,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var wg sync.WaitGroup
			wg.Add(1)
			informJobFinishCh := make(chan bool)
			resultCh := make(chan MetricsField, 1)
			go FetchContainerMetricsMean(ctx, &wg, cl, resultCh, informJobFinishCh, tt.podConfig)

			mockWaitGatlingJobRunning := func(jobFinishCh chan bool) {
				defer func() { jobFinishCh <- true }()
				time.Sleep(2 * time.Second)
			}

			mockWaitGatlingJobRunning(informJobFinishCh)

			close(informJobFinishCh)
			wg.Wait()
			close(resultCh)

			metricsResult := <-resultCh
			assert.Equal(t, tt.expected.Cpu, metricsResult.Cpu)
			assert.Equal(t, tt.expected.Memory, metricsResult.Memory)
		})
	}
}

func TestFetchTargetPodMetricsMean_Failed(t *testing.T) {
	ctx := context.TODO()

	const (
		podCpu = int64(500)                     // m vCPU
		podMem = int64(10 * 1024 * 1024 * 1024) // bytes
	)
	podLabel := map[string]string{
		"app": "sample-app",
	}
	podMetrics := &v1beta1.PodMetrics{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sample-pod",
			Namespace: "sample-namespace",
			Labels:    podLabel,
		},
		Containers: []v1beta1.ContainerMetrics{
			{
				Name: "sample-container",
				Usage: v1.ResourceList{
					v1.ResourceCPU:    resource.MustParse(fmt.Sprintf("%vm", podCpu)),
					v1.ResourceMemory: resource.MustParse(fmt.Sprintf("%v", podMem)),
				},
			},
		},
	}

	gvr := schema.GroupVersionResource{Group: "metrics.k8s.io", Version: "v1beta1", Resource: "pods"}
	cl := metricsFake.NewSimpleClientset(podMetrics)
	_ = cl.Tracker().Create(gvr, podMetrics, podMetrics.ObjectMeta.Namespace)

	resultCh := make(chan MetricsField, 1)
	informerCh := make(chan bool)

	var targetLabelKey, targetLabelVal string
	for key, val := range podMetrics.ObjectMeta.Labels {
		targetLabelKey = key
		targetLabelVal = val
	}
	targetPodConfig := cfg.TargetPodConfig{
		Namespace:     "invalid namespace",
		LabelKey:      targetLabelKey,
		LabelValue:    targetLabelVal,
		ContainerName: podMetrics.Containers[0].Name,
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go FetchContainerMetricsMean(ctx, &wg, cl, resultCh, informerCh, targetPodConfig)

	mockWaitGatlingJobRunning := func(ch chan bool) {
		defer func() { ch <- true }()
		time.Sleep(2 * time.Second)
	}

	mockWaitGatlingJobRunning(informerCh)

	close(informerCh)
	wg.Wait()
	close(resultCh)

	metricsResult, ok := <-resultCh // resultCh has no metric value so return 0 value
	assert.Equal(t, ok, false)
	assert.Equal(t, int64(0), metricsResult.Cpu)
	assert.Equal(t, int64(0), metricsResult.Memory)
}

func TestCalcAndRoundMetricsRatio(t *testing.T) {
	type input struct {
		usage int64
		limit int64
	}
	tests := []struct {
		name     string
		input    input
		expected float64
	}{
		{
			name: "receive expected value",
			input: input{
				usage: int64(499),
				limit: int64(3000),
			},
			expected: 0.166, // rounded 4th place to closest
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			roundedRatio := CalcAndRoundMetricsRatio(tt.input.usage, tt.input.limit)
			assert.Equal(t, tt.expected, roundedRatio)
		})
	}
}
