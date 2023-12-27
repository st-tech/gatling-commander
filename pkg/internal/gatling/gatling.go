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

// Package gatling implements function and type to operate gatling custom resource.
package gatling

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/st-tech/gatling-commander/pkg/util"

	gatlingv1alpha1 "github.com/st-tech/gatling-operator/api/v1alpha1"
	"gopkg.in/yaml.v3"
	kubeapiErrors "k8s.io/apimachinery/pkg/api/errors"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

/*
LoadGatlingManifest returns gatling object.

Load gatling resource manifest file and parse Bytes to Gatling object.
*/
func LoadGatlingManifest(path string) (*gatlingv1alpha1.Gatling, error) {
	gatlingYaml, _ := os.ReadFile(path)

	var (
		gatling         gatlingv1alpha1.Gatling
		gatlingManifest interface{}
	)
	if err := yaml.Unmarshal(gatlingYaml, &gatlingManifest); err != nil {
		return nil, err
	}
	// Gatling struct has json tag. so encode loaded manifest to json bytes and map field to Gatling struct object.
	jsonEncodedGatlingManifestBytes, err := json.Marshal(&gatlingManifest)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(jsonEncodedGatlingManifestBytes, &gatling); err != nil {
		return nil, err
	}
	return &gatling, nil
}

/*
CreateGatling returns error object when success to create gatling object.

If gatling object which has same name already exists in same namespace, delete old one and create new one.
*/
func CreateGatling(ctx context.Context, cl ctrlClient.Client, gatling *gatlingv1alpha1.Gatling) error {
	// check gatling object exists or not
	var foundGatling gatlingv1alpha1.Gatling
	err := cl.Get(ctx, ctrlClient.ObjectKey{
		Namespace: gatling.ObjectMeta.Namespace,
		Name:      gatling.ObjectMeta.Name,
	}, &foundGatling)
	if err != nil && !kubeapiErrors.IsNotFound(err) {
		return err
	}

	// skip delete when gatling object is not found
	if err == nil {
		if err := cl.Delete(ctx, &foundGatling); err != nil {
			return err
		}
	}

	if err := cl.Create(ctx, gatling); err != nil {
		return err
	}
	return nil
}

/*
WaitGatlingJobStartup wait until gatling job started.

Check every 5 seconds until Status.RunnerStartTime field value is set.
Except for above case, context Done or over timeout threshold, or something error occured will finish loop.
Before finish loop, cleanupGatlingJob is called and delete existing gatling object.
*/
func WaitGatlingJobStartup(
	ctx context.Context,
	cl ctrlClient.Client,
	gatling *gatlingv1alpha1.Gatling,
	timeout int32,
) error {
	var foundGatling gatlingv1alpha1.Gatling
	startTime := int32(time.Now().Unix())
	for {
		select {
		case <-ctx.Done():
			cleanupGatlingJob(cl, &foundGatling)
			return nil
		default:
			if err := cl.Get(
				ctx, ctrlClient.ObjectKey{
					Name:      gatling.ObjectMeta.Name,
					Namespace: gatling.ObjectMeta.Namespace,
				}, &foundGatling,
			); err != nil {
				return err
			}
			duration := int32(time.Now().Unix()) - startTime
			if err := util.CheckTimeout(timeout, duration); err != nil {
				cleanupGatlingJob(cl, &foundGatling)
				return err
			}
			// if RunnerStartTime to be set, finish wait loop.
			if foundGatling.Status.RunnerStartTime > 0 {
				return nil
			}
			time.Sleep(5 * time.Second)
		}
	}
}

/*
WaitGatlingJobRunning wait until gatling Job completed.

Check every 10 seconds until Status.RunnerCompleted and Status.ReportCompleted field value is set.
Check Status.RunnerStartTime field value and if its value not set (0) return error.
Except for above case, context Done or over timeout threshold, or something error occured will finish loop.
Before finish loop except for succeeded case, cleanupGatlingJob is called and delete existing gatling object.
*/
func WaitGatlingJobRunning(
	ctx context.Context,
	cl ctrlClient.Client,
	gatling *gatlingv1alpha1.Gatling,
	timeout int32,
	jobFinishCh chan bool,
) error {
	defer func() { jobFinishCh <- true }()
	var foundGatling gatlingv1alpha1.Gatling
	for {
		select {
		case <-ctx.Done(): // when interrupt
			cleanupGatlingJob(cl, &foundGatling)
			return nil
		default:
			if err := cl.Get(
				ctx, ctrlClient.ObjectKey{
					Name:      gatling.ObjectMeta.Name,
					Namespace: gatling.ObjectMeta.Namespace,
				}, &foundGatling,
			); err != nil {
				return err
			}
			if foundGatling.Status.RunnerStartTime == 0 {
				return fmt.Errorf("waitGatlingJobRunning called, but Gatling Job not started yet")
			}
			duration := int32(time.Now().Unix()) - foundGatling.Status.RunnerStartTime
			if err := util.CheckTimeout(timeout, duration); err != nil {
				cleanupGatlingJob(cl, &foundGatling)
				return err
			}
			if foundGatling.Status.RunnerCompleted && foundGatling.Status.ReportCompleted {
				fmt.Printf("Gatling Job %v completed\n", foundGatling.ObjectMeta.Name)
				return nil
			}
			time.Sleep(10 * time.Second)
		}
	}
}

/*
cleanupGatlingJob Delete specified gatling object.

Called for cleanup existing gatling object when gatling loadtest interrupted.
*/
func cleanupGatlingJob(cl ctrlClient.Client, foundGatling *gatlingv1alpha1.Gatling) {
	cleanupCtx := context.Background()
	if err := cl.Delete(cleanupCtx, foundGatling); err != nil {
		fmt.Fprintf(os.Stderr, "failed to delete found Gatling Job for cleanup, %v\n", err)
	}
}
