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

package exec

import (
	"context"
	"fmt"
	"os"
	"testing"

	cfg "github.com/st-tech/zozo-mlops-loadtest-cli/pkg/config"
	"github.com/st-tech/zozo-mlops-loadtest-cli/pkg/internal/gatling"
	gatlingTools "github.com/st-tech/zozo-mlops-loadtest-cli/pkg/internal/gatling"
	kubeutil "github.com/st-tech/zozo-mlops-loadtest-cli/pkg/internal/kubeutil"

	"github.com/google/go-cmp/cmp"
	"github.com/jinzhu/copier"
	gatlingv1alpha1 "github.com/st-tech/gatling-operator/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

const (
	ServiceName               = "sample-service"
	BaseManifest              = "testdata/base_manifest.yaml"
	Root                      = "../../.."
	ImgURL                    = "example/gatling-scenario/sample-202308021850"
	SampleGatlingManifestPath = "testdata/sample_gatling_manifest.yaml"
)

type mockCloudStorageOperator struct{}

func (op *mockCloudStorageOperator) Fetch(ctx context.Context, path string) ([]byte, error) {
	if path == "" {
		return nil, fmt.Errorf("path parameter value is empty")
	}
	data, _ := os.ReadFile(path)
	return data, nil
}

func TestLoadAndPatchBaseGatling(t *testing.T) {
	sampleGatling, err := gatlingTools.LoadGatlingManifest(SampleGatlingManifestPath)
	assert.NoError(t, err)
	tests := []struct {
		name         string
		scenarioSpec cfg.ScenarioSpec
		hasDiff      bool
	}{
		{
			name: "expected no diff",
			scenarioSpec: cfg.ScenarioSpec{
				Name: "sample-scenario",
				TestScenarioSpec: gatlingv1alpha1.TestScenarioSpec{
					SimulationClass: "SampleScenario",
					Parallelism:     1,
					Env: []corev1.EnvVar{
						{
							Name:  "ENV",
							Value: "stg",
						},
						{
							Name:  "CONCURRENCY",
							Value: "25",
						},
						{
							Name:  "DURATION",
							Value: "10",
						},
					},
				},
			},
			hasDiff: false,
		},
		{
			name: "expected diff (no env field in generated manifest)",
			scenarioSpec: cfg.ScenarioSpec{
				Name: "sample-scenario",
				TestScenarioSpec: gatlingv1alpha1.TestScenarioSpec{
					SimulationClass: "SampleScenario",
					Parallelism:     1,
				},
			},
			hasDiff: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gatling, err := loadAndPatchBaseGatling(ServiceName, ImgURL, tt.scenarioSpec, BaseManifest)
			assert.NoError(t, err)
			if diff := cmp.Diff(*sampleGatling, *gatling); (diff == "") == tt.hasDiff {
				t.Errorf("%v: unexpected diff found %v, diff %v", tt.hasDiff, diff != "", diff)
			}
		})
	}
}

func TestValidateFlags(t *testing.T) {
	// assign to var to refer to it as a pointer
	skipBuildTrue := true
	skipBuildFalse := false

	tests := []struct {
		name     string
		flags    execFlags
		config   cfg.Config
		expected error
	}{
		{
			name: "valid flag value (skipBuild true and there is imageURL value)",
			flags: execFlags{
				skipBuild: skipBuildTrue,
			},
			config: cfg.Config{
				ImageURL: "example",
			},
			expected: nil,
		},
		{
			name: "invalid flag value (skipBuild true but there is no imageURL value)",
			flags: execFlags{
				skipBuild: skipBuildTrue,
			},
			config: cfg.Config{
				ImageURL: "",
			},
			expected: fmt.Errorf("skip-build flag specified, but there is no imageURL value"),
		},
		{
			name: "valid flag value (skipBuild false and there is imageURL value)",
			flags: execFlags{
				skipBuild: skipBuildFalse,
			},
			config: cfg.Config{
				ImageURL: "example",
			},
			expected: nil,
		},
		{
			name: "valid flag value (skipBuild false and there is no imageURL value)",
			flags: execFlags{
				skipBuild: skipBuildFalse,
			},
			config: cfg.Config{
				ImageURL: "",
			},
			expected: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.flags.validateFlags(&tt.config)
			assert.Equal(t, tt.expected, err)
		})
	}
}

func TestLoadGatlingReportFromCloudStorage(t *testing.T) {
	cl := kubeutil.InitFakeClient()
	reportCompletedGatling, err := gatlingTools.LoadGatlingManifest(SampleGatlingManifestPath)
	assert.NoError(t, err)
	reportCompletedGatling.Status = gatlingv1alpha1.GatlingStatus{
		ReportCompleted:   true,
		ReportStoragePath: "testdata/gatling_report_sample",
	}
	// create Status.ReportCompleted field value false gatling object
	err = cl.Create(context.TODO(), reportCompletedGatling)
	assert.NoError(t, err)
	op := &mockCloudStorageOperator{}
	tests := []struct {
		name    string
		gatling *gatlingv1alpha1.Gatling
	}{
		{
			name:    "load gatling report from cloud storage success",
			gatling: reportCompletedGatling,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := loadGatlingReportFromCloudStorage(context.TODO(), op, cl, tt.gatling)
			assert.NoError(t, err)
		})
	}
}

func TestLoadGatlingReportFromCloudStorage_Fail(t *testing.T) {
	cl := kubeutil.InitFakeClient()
	reportCompletedFalseGatling, err := gatlingTools.LoadGatlingManifest(SampleGatlingManifestPath)
	assert.NoError(t, err)
	noReportStoragePathGatling, err := gatlingTools.LoadGatlingManifest(SampleGatlingManifestPath)
	assert.NoError(t, err)
	reportCompletedFalseGatling.ObjectMeta.Name = "report completed"
	reportCompletedFalseGatling.Status = gatlingv1alpha1.GatlingStatus{
		ReportCompleted:   false,
		ReportStoragePath: "",
	}
	noReportStoragePathGatling.ObjectMeta.Name = "no report storage path"
	noReportStoragePathGatling.Status = gatlingv1alpha1.GatlingStatus{
		ReportCompleted:   true,
		ReportStoragePath: "",
	}
	// create gatling object preparation
	err = cl.Create(context.TODO(), reportCompletedFalseGatling)
	assert.NoError(t, err)
	err = cl.Create(context.TODO(), noReportStoragePathGatling)
	assert.NoError(t, err)
	op := &mockCloudStorageOperator{}
	tests := []struct {
		name    string
		gatling *gatlingv1alpha1.Gatling
	}{
		{
			name:    "gatling object not exists",
			gatling: &gatlingv1alpha1.Gatling{},
		},
		{
			name:    "gatling report completed false",
			gatling: reportCompletedFalseGatling,
		},
		{
			name:    "gatling report storage path field is invalid",
			gatling: noReportStoragePathGatling,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := loadGatlingReportFromCloudStorage(context.TODO(), op, cl, tt.gatling)
			assert.Error(t, err)
		})
	}
}

func TestCheckContinueToExec_FailFast(t *testing.T) {
	var failedExistsReport gatling.GatlingReport
	sampleReport := gatling.GatlingReport{
		FiftiethPercentiles: gatling.GatlingReportStats{
			Ok: 70,
		},
		SeventyFifthPercentiles: gatling.GatlingReportStats{
			Ok: 80,
		},
		NintyFifthPercentiles: gatling.GatlingReportStats{
			Ok: 90,
		},
		NintyNinthPercentiles: gatling.GatlingReportStats{
			Ok: 100,
		},
	}
	err := copier.CopyWithOption(&failedExistsReport, sampleReport, copier.Option{
		IgnoreEmpty: false,
		DeepCopy:    true,
	})
	assert.NoError(t, err)
	failedExistsReport.Failed.Percentage = 10

	tests := []struct {
		name          string
		serviceConfig serviceConfig
		gatlingReport gatling.GatlingReport
		expected      *checkContinueToExecResult
	}{
		{
			name: "failFast false, should continue",
			serviceConfig: serviceConfig{
				failFast: false,
			},
			gatlingReport: sampleReport,
			expected: &checkContinueToExecResult{
				shouldContinue: true,
				message:        "",
			},
		},
		{
			name: "failFast true, should not continue",
			serviceConfig: serviceConfig{
				failFast: true,
			},
			gatlingReport: failedExistsReport,
			expected: &checkContinueToExecResult{
				shouldContinue: false,
				message:        "failed percentage greater than 0",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checkContinue, err := checkContinueToExec(tt.serviceConfig, tt.gatlingReport)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, checkContinue)
		})
	}
}

func TestCheckContinueToExec_TargetLatency(t *testing.T) {
	sampleReport := gatling.GatlingReport{
		FiftiethPercentiles: gatling.GatlingReportStats{
			Ok: 70,
		},
		SeventyFifthPercentiles: gatling.GatlingReportStats{
			Ok: 80,
		},
		NintyFifthPercentiles: gatling.GatlingReportStats{
			Ok: 90,
		},
		NintyNinthPercentiles: gatling.GatlingReportStats{
			Ok: 100,
		},
		Failed: gatling.GatlingReportGroup{
			Percentage: 0,
		},
	}

	latencyBelowServiceConfig := serviceConfig{
		failFast:         true,
		targetLatency:    10,
		targetPercentile: 99,
	}

	tests := []struct {
		name          string
		serviceConfig serviceConfig
		gatlingReport gatling.GatlingReport
		expected      *checkContinueToExecResult
	}{
		{
			name: "check latency, should continue",
			serviceConfig: serviceConfig{
				failFast:         true,
				targetLatency:    800,
				targetPercentile: 99,
			},
			gatlingReport: sampleReport,
			expected: &checkContinueToExecResult{
				shouldContinue: true,
				message:        "",
			},
		},
		{
			name: "not check latency, should continue",
			serviceConfig: serviceConfig{
				failFast:         true,
				targetLatency:    0,
				targetPercentile: 0,
			},
			gatlingReport: sampleReport,
			expected: &checkContinueToExecResult{
				shouldContinue: true,
				message:        "",
			},
		},
		{
			name:          "check latency, should not continue",
			serviceConfig: latencyBelowServiceConfig,
			gatlingReport: sampleReport,
			expected: &checkContinueToExecResult{
				shouldContinue: false,
				message: fmt.Sprintf(
					"latency below specified target, target: %v, result: %v",
					latencyBelowServiceConfig.targetLatency,
					sampleReport.NintyNinthPercentiles.Ok,
				),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checkContinue, err := checkContinueToExec(tt.serviceConfig, sampleReport)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, checkContinue)
		})
	}
}

func TestCheckContinueToExec_Failed(t *testing.T) {
	sampleReport := gatling.GatlingReport{
		FiftiethPercentiles: gatling.GatlingReportStats{
			Ok: 70,
		},
		SeventyFifthPercentiles: gatling.GatlingReportStats{
			Ok: 80,
		},
		NintyFifthPercentiles: gatling.GatlingReportStats{
			Ok: 90,
		},
		NintyNinthPercentiles: gatling.GatlingReportStats{
			Ok: 100,
		},
		Failed: gatling.GatlingReportGroup{
			Percentage: 0,
		},
	}

	tests := []struct {
		name          string
		serviceConfig serviceConfig
		gatlingReport gatling.GatlingReport
		expected      error
	}{
		{
			name: "error invalid percentile specified",
			serviceConfig: serviceConfig{
				failFast:         true,
				targetLatency:    800,
				targetPercentile: 80,
			},
			gatlingReport: sampleReport,
			expected: fmt.Errorf(
				"failed to get specified percentile latency %v",
				fmt.Errorf("specified percentile value is not matched to GatlingReport field"),
			),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := checkContinueToExec(tt.serviceConfig, sampleReport)
			assert.Equal(t, tt.expected, err)
		})
	}
}
