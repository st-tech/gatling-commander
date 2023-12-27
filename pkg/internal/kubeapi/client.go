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
	gatlingv1alpha1 "github.com/st-tech/gatling-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	metricsClientset "k8s.io/metrics/pkg/client/clientset/versioned"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlConfig "sigs.k8s.io/controller-runtime/pkg/client/config"
)

/*
InitClient returns client of kubeapi.

use for operate gatling object.
*/
func InitClient(k8sCtxName string) (ctrlClient.Client, error) {
	// add custom resource gatling to scheme
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(gatlingv1alpha1.AddToScheme(scheme))

	k8sConfig, err := ctrlConfig.GetConfigWithContext(k8sCtxName)
	if err != nil {
		return nil, err
	}
	cl, err := ctrlClient.New(k8sConfig, ctrlClient.Options{
		Scheme: scheme,
	})
	if err != nil {
		return nil, err
	}
	return cl, nil
}

// InitMetricsClient returns clientset of metrics.
func InitMetricsClient(k8sCtxName string) (*metricsClientset.Clientset, error) {
	k8sConfig, err := ctrlConfig.GetConfigWithContext(k8sCtxName)
	if err != nil {
		return nil, err
	}
	cl, err := metricsClientset.NewForConfig(k8sConfig)
	if err != nil {
		return nil, err
	}
	return cl, nil
}
