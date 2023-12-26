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

/*
Package config map config.yaml to config.Config object and other nested struct type object.
*/
package config

import (
	"fmt"

	"github.com/st-tech/zozo-mlops-loadtest-cli/pkg/internal/gatling"
	"github.com/st-tech/zozo-mlops-loadtest-cli/pkg/util"
)

// Config map config/config.yaml field value.
type Config struct {
	GatlingContextName   string      `yaml:"gatlingContextName"`
	ImageRepository      string      `yaml:"imageRepository"`
	ImagePrefix          string      `yaml:"imagePrefix"`
	ImageURL             string      `yaml:"imageURL"`
	GatlingDockerfileDir string      `yaml:"gatlingDockerfileDir"`
	BaseManifest         string      `yaml:"baseManifest"`
	StartupTimeoutSec    int32       `yaml:"startupTimeoutSec"`
	ExecTimeoutSec       int32       `yaml:"execTimeoutSec"`
	SlackConfig          SlackConfig `yaml:"slackConfig"`
	Services             []Service   `yaml:"services"`
}

/*
ValidateFieldValue validate config/config.yaml field value.

Check items are below.
  - each of Config object field value is set
  - each of Service object field required value is set
  - each of TargetPodConfig object is valid
  - Service objects TargetPercentile and TargetLatency fields value are valid
*/
func (c *Config) ValidateFieldValue() error {
	if c.GatlingContextName == "" {
		return fmt.Errorf("config param gatlingContextName is required")
	}
	if c.ImageRepository == "" {
		return fmt.Errorf("config param imageRepostory is required")
	}
	if c.ImagePrefix == "" {
		return fmt.Errorf("config param imagePrefix is required")
	}
	if c.GatlingDockerfileDir == "" {
		return fmt.Errorf("config param gatlingDockerfileDir is required")
	}
	if c.BaseManifest == "" {
		return fmt.Errorf("config param baseManifest is required")
	}
	if c.StartupTimeoutSec == 0 {
		return fmt.Errorf("config param startupTimeout is required")
	}
	if c.ExecTimeoutSec == 0 {
		return fmt.Errorf("config param execTimeout is required")
	}
	serviceNames := make([]string, 0, len(c.Services))
	for _, service := range c.Services {
		if service.Name == "" {
			return fmt.Errorf("config param service[].name is required")
		}
		if service.SpreadsheetId == "" {
			return fmt.Errorf("config param service[].spreadsheetID is required")
		}
		err := validateGetTargetPodRequiredField(service.TargetPodConfig)
		if err != nil {
			return fmt.Errorf("config param filter target pod param is invalid %v", err)
		}
		err = validateTargetLatencyField(service.TargetPercentile, service.TargetLatency)
		if err != nil {
			return fmt.Errorf("config param check latency field value is invalid %v", err)
		}
		serviceNames = append(serviceNames, service.Name)
		scenarioSpecNames := make([]string, 0, len(service.ScenarioSpecs))
		for _, scenarioSpec := range service.ScenarioSpecs {
			scenarioSpecNames = append(scenarioSpecNames, scenarioSpec.Name+scenarioSpec.SubName)
		}
		if err := util.CheckDuplicate(scenarioSpecNames); err != nil {
			return fmt.Errorf("%v, config.yaml scenarioSpec name duplicated in service %v", err, service.Name)
		}
	}
	if err := util.CheckDuplicate(serviceNames); err != nil {
		return fmt.Errorf("%v config.yaml service name duplicated", err)
	}
	return nil
}

// validateTargetLatencyField validate config.yaml target latency field value.
func validateTargetLatencyField(percentile uint32, latency float64) error {
	if latency < float64(0) {
		return fmt.Errorf("invalid latency value specified, it must be more than 0")
	}
	err := checkTargetLatencyRequiredField(percentile, latency)
	if err != nil {
		return err
	}
	return nil
}

// checkTargetLatencyRequiredField check required field for check target latency.
func checkTargetLatencyRequiredField(percentile uint32, latency float64) error {
	if percentile == 0 && latency == 0 { // case: config.yaml targetPercentile & targetLatency value is empty
		return nil
	}

	if percentile == 0 || latency == 0 {
		return fmt.Errorf("percentile must be set with latency, one of these is empty")
	}
	gatlingReport := &gatling.GatlingReport{}
	if _, err := gatlingReport.GetPercentileLatency(percentile); err != nil { // validate specified percentile value
		return err
	}
	return nil
}

/*
validateGetTargetPodRequiredField validate config.yaml targetPodConfig field value.

Check items are below.
  - each of podConfig field value is set
*/
func validateGetTargetPodRequiredField(podConfig TargetPodConfig) error {
	if podConfig.ContextName == "" {
		return fmt.Errorf("targetPod field contextName is required")
	}
	if podConfig.Namespace == "" {
		return fmt.Errorf("targetPod field namespace is required")
	}
	if podConfig.LabelKey == "" {
		return fmt.Errorf("targetPod field podLabelKey is required")
	}
	if podConfig.LabelValue == "" {
		return fmt.Errorf("targetPod field podLabelValue is required")
	}
	if podConfig.ContainerName == "" {
		return fmt.Errorf("targetPod field containerName is required")
	}
	return nil
}
