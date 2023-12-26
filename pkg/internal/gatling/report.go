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
	"encoding/json"
	"fmt"
	"strconv"

	gatlingv1alpha1 "github.com/st-tech/gatling-operator/api/v1alpha1"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

/*
GatlingReport has field of gatling report.

refer to gatling report json key "/js/global_stats.json".
*/
type GatlingReport struct {
	Name                                                   string             `json:"name"`
	NumberOfRequests                                       GatlingReportStats `json:"numberOfRequests"`
	MinResponseTime                                        GatlingReportStats `json:"minResponseTime"`
	MaxResponseTime                                        GatlingReportStats `json:"maxResponseTime"`
	MeanResponseTime                                       GatlingReportStats `json:"meanResponseTime"`
	StandardDeviation                                      GatlingReportStats `json:"standardDeviation"`
	FiftiethPercentiles                                    GatlingReportStats `json:"percentiles1"`
	SeventyFifthPercentiles                                GatlingReportStats `json:"percentiles2"`
	NintyFifthPercentiles                                  GatlingReportStats `json:"percentiles3"`
	NintyNinthPercentiles                                  GatlingReportStats `json:"percentiles4"`
	MeanNumberOfRequestsPerSecond                          GatlingReportStats `json:"meanNumberOfRequestsPerSecond"`
	UnderEightHundredMilliSec                              GatlingReportGroup `json:"group1"`
	BetweenFromEightHundredToOneThousandTwoHundredMilliSec GatlingReportGroup `json:"group2"`
	OverOneThousandTwoHundredMilliSec                      GatlingReportGroup `json:"group3"`
	Failed                                                 GatlingReportGroup `json:"group4"`
}

// GetPercentileLatency get latency match to specified percentile.
func (r *GatlingReport) GetPercentileLatency(percentile uint32) (float64, error) {
	var latency float64
	switch percentile {
	case 99:
		latency = r.NintyNinthPercentiles.Ok
	case 95:
		latency = r.NintyFifthPercentiles.Ok
	case 75:
		latency = r.SeventyFifthPercentiles.Ok
	case 50:
		latency = r.FiftiethPercentiles.Ok
	default:
		return 0, fmt.Errorf("specified percentile value is not matched to GatlingReport field")
	}
	return latency, nil
}

// BytesToGatlingReport parse jsonBytes to GatlingReport object.
func BytesToGatlingReport(jsonBytes []byte) (*GatlingReport, error) {
	var gatlingReport GatlingReport
	if err := json.Unmarshal(jsonBytes, &gatlingReport); err != nil {
		return &GatlingReport{}, err
	}
	return &gatlingReport, nil
}

/*
ExtractLoadtestConditionToReport extract gatling object TestScenarioSpec field value and returns them in a format
matching the spreadsheet columns.

Extract DURATION and CONCURRENCY value, and the values of the other fields are summarized in the same column.
*/
func ExtractLoadtestConditionToReport(
	testScenarioSpec gatlingv1alpha1.TestScenarioSpec,
) (concurrency string, duration string, condition string, err error) {
	for _, field := range testScenarioSpec.Env {
		switch field.Name {
		case "DURATION":
			duration = field.Value
		case "CONCURRENCY":
			singleConcurrency, err := strconv.Atoi(field.Value)
			if err != nil {
				return "", "", "", err
			}
			concurrency = fmt.Sprintf("%v", testScenarioSpec.Parallelism*int32(singleConcurrency))
		default:
			condition += fmt.Sprintf("%v=%v,", field.Name, field.Value)
		}
	}
	return concurrency, duration, condition, nil
}

// GetGatlingReportStoragePath fetch gatling object and get ReportStoragePath value.
func GetGatlingReportStoragePath(
	ctx context.Context,
	cl ctrlClient.Client,
	gatling *gatlingv1alpha1.Gatling,
) (string, error) {
	var foundGatling gatlingv1alpha1.Gatling
	if err := cl.Get(
		ctx, ctrlClient.ObjectKey{
			Name:      gatling.ObjectMeta.Name,
			Namespace: gatling.ObjectMeta.Namespace,
		}, &foundGatling,
	); err != nil {
		return "", err
	}

	if !foundGatling.Status.ReportCompleted {
		return "", fmt.Errorf(
			"found gatling object status.ReportCompleted field value is not true %v\n",
			foundGatling.Status.ReportCompleted,
		)
	}
	return foundGatling.Status.ReportStoragePath, nil
}
