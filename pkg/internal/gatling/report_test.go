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

	"github.com/st-tech/zozo-mlops-loadtest-cli/pkg/internal/kubeutil"

	gatlingv1alpha1 "github.com/st-tech/gatling-operator/api/v1alpha1"
	"github.com/stretchr/testify/assert"
)

func TestGetPercentileLatency_Success(t *testing.T) {
	sampleReport := &GatlingReport{
		FiftiethPercentiles: GatlingReportStats{
			Ok: 70,
		},
		SeventyFifthPercentiles: GatlingReportStats{
			Ok: 80,
		},
		NintyFifthPercentiles: GatlingReportStats{
			Ok: 90,
		},
		NintyNinthPercentiles: GatlingReportStats{
			Ok: 100,
		},
	}
	tests := []struct {
		name     string
		input    uint32
		expected float64
	}{
		{
			name:     "specify 99 percentile",
			input:    99,
			expected: sampleReport.NintyNinthPercentiles.Ok,
		},
		{
			name:     "specify 95 percentile",
			input:    95,
			expected: sampleReport.NintyFifthPercentiles.Ok,
		},
		{
			name:     "specify 75 percentile",
			input:    75,
			expected: sampleReport.SeventyFifthPercentiles.Ok,
		},
		{
			name:     "specify 50 percentile",
			input:    50,
			expected: sampleReport.FiftiethPercentiles.Ok,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			latency, err := sampleReport.GetPercentileLatency(tt.input)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, latency)
		})
	}
}

func TestGetPercentileLatency_Fail(t *testing.T) {
	SampleReport := &GatlingReport{
		FiftiethPercentiles: GatlingReportStats{
			Ok: 70,
		},
		SeventyFifthPercentiles: GatlingReportStats{
			Ok: 80,
		},
		NintyFifthPercentiles: GatlingReportStats{
			Ok: 90,
		},
		NintyNinthPercentiles: GatlingReportStats{
			Ok: 100,
		},
	}
	tests := []struct {
		name     string
		input    uint32
		expected error
	}{
		{
			name:     "invalid percentile value",
			input:    80,
			expected: fmt.Errorf("specified percentile value is not matched to GatlingReport field"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			latency, err := SampleReport.GetPercentileLatency(tt.input)
			assert.Equal(t, tt.expected, err)
			assert.Equal(t, float64(0), latency)
		})
	}
}

func TestGetGatlingReportStoragePath(t *testing.T) {
	cl := kubeutil.InitFakeClient()
	reportCompletedGatling, err := LoadGatlingManifest(SampleGatlingManifestPath)
	assert.NoError(t, err)
	reportCompletedGatling.Status = gatlingv1alpha1.GatlingStatus{
		ReportCompleted:   true,
		ReportStoragePath: "gs://test-bucket/sample-reports/99999999",
	}
	// create Status.ReportCompleted field value false gatling object
	err = cl.Create(context.TODO(), reportCompletedGatling)
	assert.NoError(t, err)

	tests := []struct {
		name    string
		gatling *gatlingv1alpha1.Gatling
	}{
		{
			name:    "report completed and have report storage path field value",
			gatling: reportCompletedGatling,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reportStoragePath, err := GetGatlingReportStoragePath(context.TODO(), cl, tt.gatling)
			assert.Equal(t, reportStoragePath, tt.gatling.Status.ReportStoragePath)
			assert.NoError(t, err)
		})
	}
}

func TestGetGatlingReportStoragePath_ReportCompletedNotTrue(t *testing.T) {
	cl := kubeutil.InitFakeClient()
	reportCompletedFalseGatling, err := LoadGatlingManifest(SampleGatlingManifestPath)
	assert.NoError(t, err)
	reportCompletedFalseGatling.Status = gatlingv1alpha1.GatlingStatus{
		ReportCompleted:   false,
		ReportStoragePath: "gs://test-bucket/sample-reports/99999999",
	}
	// create Status.ReportCompleted field value false gatling object
	err = cl.Create(context.TODO(), reportCompletedFalseGatling)
	assert.NoError(t, err)

	tests := []struct {
		name    string
		gatling *gatlingv1alpha1.Gatling
	}{
		{
			name:    "report completed false",
			gatling: reportCompletedFalseGatling,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := GetGatlingReportStoragePath(context.TODO(), cl, tt.gatling)
			assert.Error(t, err)
		})
	}
}
