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

package gatling

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/st-tech/gatling-commander/pkg/internal/kubeutil"

	gatlingv1alpha1 "github.com/st-tech/gatling-operator/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/stretchr/testify/assert"
)

const (
	SampleGatlingManifestPath = "testdata/sample_gatling_manifest.yaml"
)

func TestCreateGatling(t *testing.T) {
	cl := kubeutil.InitFakeClient()

	// prepare gatling data for test
	sampleGatling, err := LoadGatlingManifest(SampleGatlingManifestPath)
	assert.NoError(t, err)

	existsGatling, notExistsGatling := *sampleGatling, *sampleGatling
	notExistsGatling.ObjectMeta.Name = "new-gatling"

	// create sample gatling object for existsGatling test
	err = cl.Create(context.TODO(), sampleGatling)
	assert.NoError(t, err)

	tests := []struct {
		name    string
		gatling *gatlingv1alpha1.Gatling
	}{
		{
			name:    "create new gatling object",
			gatling: &notExistsGatling,
		},
		{
			name:    "delete existing and create new gatling object",
			gatling: &existsGatling,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := testCreateGatling(cl, tt.gatling)
			assert.NoError(t, err)
		})
	}
}

func TestWaitGatlingJobStartup(t *testing.T) {
	cl := kubeutil.InitFakeClient()
	startedGatling, err := LoadGatlingManifest(SampleGatlingManifestPath)
	assert.NoError(t, err)

	startedGatling.Status = gatlingv1alpha1.GatlingStatus{
		RunnerStartTime: int32(time.Now().Unix()),
	}
	err = cl.Create(context.TODO(), startedGatling)
	assert.NoError(t, err)

	tests := []struct {
		name    string
		gatling *gatlingv1alpha1.Gatling
	}{
		{
			name:    "wait found gatling startup",
			gatling: startedGatling,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := WaitGatlingJobStartup(context.TODO(), cl, tt.gatling, 2)
			assert.NoError(t, err)
		})
	}
}

func TestWaitGatlingJobStartup_ExpectedFail(t *testing.T) {
	cl := kubeutil.InitFakeClient()

	noRunnerStartTimeGatling, err := LoadGatlingManifest(SampleGatlingManifestPath)
	assert.NoError(t, err)
	noRunnerStartTimeGatling.Status = gatlingv1alpha1.GatlingStatus{
		RunnerStartTime: 0,
	}

	err = cl.Create(context.TODO(), noRunnerStartTimeGatling)
	assert.NoError(t, err)

	tests := []struct {
		name     string
		gatling  *gatlingv1alpha1.Gatling
		timeout  int32
		expected error
	}{
		{
			name:     "wait startup failed with timeout",
			gatling:  noRunnerStartTimeGatling,
			timeout:  1,
			expected: fmt.Errorf("timeout %v execeeded", 1),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := WaitGatlingJobStartup(context.TODO(), cl, tt.gatling, tt.timeout)
			assert.Equal(t, tt.expected, err)
		})
	}
}

func TestWaitGatlingJobRunning(t *testing.T) {
	cl := kubeutil.InitFakeClient()

	completedGatling, err := LoadGatlingManifest(SampleGatlingManifestPath)
	assert.NoError(t, err)

	completedGatling.Status = gatlingv1alpha1.GatlingStatus{
		RunnerStartTime: int32(time.Now().Unix()),
		RunnerCompleted: true,
		ReportCompleted: true,
	}
	err = cl.Create(context.TODO(), completedGatling)
	assert.NoError(t, err)

	informJobFinishCh := make(chan bool, 1)

	tests := []struct {
		name    string
		gatling *gatlingv1alpha1.Gatling
	}{
		{
			name:    "wait found gatling completed",
			gatling: completedGatling,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := WaitGatlingJobRunning(context.TODO(), cl, tt.gatling, 2, informJobFinishCh)
			assert.NoError(t, err)
		})
	}
}

func TestWaitGatlingJobRunning_ExpectedFail(t *testing.T) {
	cl := kubeutil.InitFakeClient()

	sampleGatling, err := LoadGatlingManifest(SampleGatlingManifestPath)
	assert.NoError(t, err)

	sampleGatling.Status = gatlingv1alpha1.GatlingStatus{
		RunnerStartTime: int32(time.Now().Unix()),
	}
	existsGatling, notExistsGatling, noRunnerStartTimeGatling := *sampleGatling, *sampleGatling, *sampleGatling
	notExistsGatling.ObjectMeta.Name = "notexists"
	noRunnerStartTimeGatling.Status.RunnerStartTime = 0

	err = cl.Create(context.TODO(), &existsGatling)
	assert.NoError(t, err)

	tests := []struct {
		name       string
		gatling    *gatlingv1alpha1.Gatling
		timeout    int32
		informerCh chan bool
	}{
		{
			name:       "handle not found error",
			gatling:    &notExistsGatling,
			timeout:    1,
			informerCh: make(chan bool, 1),
		},
		{
			name:       "handle timeout error",
			gatling:    &existsGatling,
			timeout:    0,
			informerCh: make(chan bool, 1),
		},
		{
			name:       "Gatling Job not started",
			gatling:    &noRunnerStartTimeGatling,
			timeout:    1,
			informerCh: make(chan bool, 1),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := WaitGatlingJobRunning(context.TODO(), cl, tt.gatling, tt.timeout, tt.informerCh)
			assert.Error(t, err)
		})
	}
}

func TestWaitGatlingJobRunning_CleanupGatlingJobWhenInterrupted(t *testing.T) {
	cl := kubeutil.InitFakeClient()

	notCompletedGatling, err := LoadGatlingManifest(SampleGatlingManifestPath)
	assert.NoError(t, err)

	notCompletedGatling.Status = gatlingv1alpha1.GatlingStatus{
		RunnerStartTime: int32(time.Now().Unix()),
		RunnerCompleted: false,
		ReportCompleted: false,
	}
	err = cl.Create(context.TODO(), notCompletedGatling)
	assert.NoError(t, err)

	informJobFinishCh := make(chan bool, 1)

	tests := []struct {
		name    string
		gatling *gatlingv1alpha1.Gatling
	}{
		{
			name:    "cleanup gatling job succeeded",
			gatling: notCompletedGatling,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			go func() {
				time.Sleep(1 * time.Second)
				cancel()
			}()
			err := WaitGatlingJobRunning(ctx, cl, tt.gatling, 2, informJobFinishCh)
			assert.NoError(t, err)
			var foundGatling gatlingv1alpha1.Gatling
			err = cl.Get(
				context.TODO(),
				ctrlClient.ObjectKey{
					Namespace: notCompletedGatling.ObjectMeta.Namespace,
					Name:      notCompletedGatling.ObjectMeta.Name,
				},
				&foundGatling,
			)
			fmt.Printf("%v\n", err)
		})
	}
}

func testCreateGatling(cl client.Client, newGatling *gatlingv1alpha1.Gatling) error {
	err := CreateGatling(context.TODO(), cl, newGatling)
	if err != nil {
		return err
	}

	var foundGatling gatlingv1alpha1.Gatling
	err = cl.Get(
		context.TODO(),
		ctrlClient.ObjectKey{
			Namespace: newGatling.ObjectMeta.Namespace,
			Name:      newGatling.ObjectMeta.Name,
		},
		&foundGatling,
	)
	if err != nil {
		return fmt.Errorf("unexpected error occured when get gatling object %v", err)
	}
	return nil
}
