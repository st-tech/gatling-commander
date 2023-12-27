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

package config

import (
	"fmt"
	"os"
	"testing"

	"github.com/jinzhu/copier"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

type targetLatencyField struct {
	percentile uint32
	latency    float64
}

var validConfig Config

func init() {
	testing.Init()
	validConfigYaml, _ := os.ReadFile("testdata/valid_config.yaml")
	if err := yaml.Unmarshal(validConfigYaml, &validConfig); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func TestValidateFieldValue(t *testing.T) {
	noContextNameField, noImgRepoField, noImgPrefixField := validConfig, validConfig, validConfig
	noGatlingDockerfileDirField, noBaseManifestField, noStartupTimeoutSecField := validConfig, validConfig, validConfig
	noExecTimeoutSecField, serviceNameDuplicate := validConfig, validConfig
	var (
		noServiceNameField        Config
		noSpreadsheetIdField      Config
		scenarioSpecNameDuplicate Config
	)
	err := copier.CopyWithOption(&noServiceNameField, validConfig, copier.Option{
		IgnoreEmpty: false,
		DeepCopy:    true,
	})
	assert.NoError(t, err)
	err = copier.CopyWithOption(&noSpreadsheetIdField, validConfig, copier.Option{
		IgnoreEmpty: false,
		DeepCopy:    true,
	})
	assert.NoError(t, err)
	err = copier.CopyWithOption(&scenarioSpecNameDuplicate, validConfig, copier.Option{
		IgnoreEmpty: false,
		DeepCopy:    true,
	})
	assert.NoError(t, err)

	noContextNameField.GatlingContextName = ""
	noImgRepoField.ImageRepository = ""
	noImgPrefixField.ImagePrefix = ""
	noGatlingDockerfileDirField.GatlingDockerfileDir = ""
	noBaseManifestField.BaseManifest = ""
	noStartupTimeoutSecField.StartupTimeoutSec = 0
	noExecTimeoutSecField.ExecTimeoutSec = 0
	noServiceNameField.Services[0].Name = ""
	noSpreadsheetIdField.Services[0].SpreadsheetId = ""
	serviceNameDuplicate.Services = append(serviceNameDuplicate.Services, serviceNameDuplicate.Services[0])
	duplicateServiceName := serviceNameDuplicate.Services[0].Name
	serviceNameDuplicateErr := fmt.Errorf("duplicated value found %v\n", []string{duplicateServiceName})
	scenarioSpecNameDuplicate.Services[0].ScenarioSpecs = append(
		scenarioSpecNameDuplicate.Services[0].ScenarioSpecs,
		scenarioSpecNameDuplicate.Services[0].ScenarioSpecs[0],
	)
	scenarioSpecNameDuplicate.Services[0].ScenarioSpecs[1].Name = scenarioSpecNameDuplicate.Services[0].ScenarioSpecs[0].Name       // nolint:lll
	scenarioSpecNameDuplicate.Services[0].ScenarioSpecs[1].SubName = scenarioSpecNameDuplicate.Services[0].ScenarioSpecs[0].SubName // nolint:lll
	scenarioSpecNameDuplicateErr := fmt.Errorf(
		"duplicated value found %v\n",
		[]string{
			scenarioSpecNameDuplicate.Services[0].ScenarioSpecs[0].Name + scenarioSpecNameDuplicate.Services[0].ScenarioSpecs[0].SubName, // nolint:lll
		},
	)

	tests := []struct {
		name     string
		config   Config
		expected error
	}{
		{
			name:     "valid config field value",
			config:   validConfig,
			expected: nil,
		},
		{
			name:     "lack of config gatlingContextName field value",
			config:   noContextNameField,
			expected: fmt.Errorf("config param gatlingContextName is required"),
		},
		{
			name:     "lack of config imageRepository field value",
			config:   noImgRepoField,
			expected: fmt.Errorf("config param imageRepostory is required"),
		},
		{
			name:     "lack of config imagePrefix field value",
			config:   noImgPrefixField,
			expected: fmt.Errorf("config param imagePrefix is required"),
		},
		{
			name:     "lack of config gatlingDockerfileDir field value",
			config:   noGatlingDockerfileDirField,
			expected: fmt.Errorf("config param gatlingDockerfileDir is required"),
		},
		{
			name:     "lack of config baseManifest field value",
			config:   noBaseManifestField,
			expected: fmt.Errorf("config param baseManifest is required"),
		},
		{
			name:     "lack of config startupTimeoutSec field value",
			config:   noStartupTimeoutSecField,
			expected: fmt.Errorf("config param startupTimeout is required"),
		},
		{
			name:     "lack of config execTimeoutSec field value",
			config:   noExecTimeoutSecField,
			expected: fmt.Errorf("config param execTimeout is required"),
		},
		{
			name:     "lack of config service name field value",
			config:   noServiceNameField,
			expected: fmt.Errorf("config param service[].name is required"),
		},
		{
			name:     "lack of config service spreadsheetId field value",
			config:   noSpreadsheetIdField,
			expected: fmt.Errorf("config param service[].spreadsheetID is required"),
		},
		{
			name:     "config services[].name field value duplicate",
			config:   serviceNameDuplicate,
			expected: fmt.Errorf("%v config.yaml service name duplicated", serviceNameDuplicateErr),
		},
		{
			name:   "config services[].scenarioSpecs[].name field value duplicate",
			config: scenarioSpecNameDuplicate,
			expected: fmt.Errorf(
				"%v, config.yaml scenarioSpec name duplicated in service %v",
				scenarioSpecNameDuplicateErr,
				duplicateServiceName,
			),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.ValidateFieldValue()
			assert.Equal(t, tt.expected, err)
		})
	}
}

func TestValidateTargetLatencyField_Success(t *testing.T) {
	tests := []struct {
		name  string
		input targetLatencyField
	}{
		{
			name: "percentile and latency is not specified (0)",
			input: targetLatencyField{
				percentile: 0,
				latency:    0,
			},
		},
		{
			name: "percentile and latency is specified",
			input: targetLatencyField{
				percentile: 99,
				latency:    100,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTargetLatencyField(tt.input.percentile, tt.input.latency)
			assert.NoError(t, err)
		})
	}
}

func TestValidateTargetLatencyField_Failed(t *testing.T) {
	tests := []struct {
		name     string
		input    targetLatencyField
		expected error
	}{
		{
			name: "percentile and latency is negative value",
			input: targetLatencyField{
				percentile: 99,
				latency:    -100,
			},
			expected: fmt.Errorf("invalid latency value specified, it must be more than 0"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTargetLatencyField(tt.input.percentile, tt.input.latency)
			assert.Equal(t, tt.expected, err)
		})
	}
}

func TestCheckTargetLatencyRequiredField_Success(t *testing.T) {
	tests := []struct {
		name  string
		input targetLatencyField
	}{
		{
			name: "percentile and latency is not specified (0)",
			input: targetLatencyField{
				percentile: 0,
				latency:    0,
			},
		},
		{
			name: "percentile and latency is specified",
			input: targetLatencyField{
				percentile: 99,
				latency:    100,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTargetLatencyField(tt.input.percentile, tt.input.latency)
			assert.NoError(t, err)
		})
	}
}

func TestCheckTargetLatencyRequiredField_Failed(t *testing.T) {
	tests := []struct {
		name     string
		input    targetLatencyField
		expected error
	}{
		{
			name: "percentile and latency is negative value",
			input: targetLatencyField{
				percentile: 99,
				latency:    -100,
			},
			expected: fmt.Errorf("invalid latency value specified, it must be more than 0"),
		},
		{
			name: "target percentile or target latency is not specified",
			input: targetLatencyField{
				percentile: 0,
				latency:    100,
			},
			expected: fmt.Errorf("percentile must be set with latency, one of these is empty"),
		},
		{
			name: "invalid percentile specified",
			input: targetLatencyField{
				percentile: 80,
				latency:    100,
			},
			expected: fmt.Errorf("specified percentile value is not matched to GatlingReport field"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTargetLatencyField(tt.input.percentile, tt.input.latency)
			assert.Equal(t, tt.expected, err)
		})
	}
}

func TestValidateGetTargetPodRequiredField(t *testing.T) {
	tests := []struct {
		name  string
		input TargetPodConfig
	}{
		{
			name: "validate get target pod required field success",
			input: TargetPodConfig{
				ContextName:   "gke_sample_asia-east1_gke_sample_asia",
				Namespace:     "sample",
				LabelKey:      "run",
				LabelValue:    "sample-api",
				ContainerName: "sample-api",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateGetTargetPodRequiredField(tt.input)
			assert.NoError(t, err)
		})
	}
}

func TestValidateGetTargetPodRequiredField_Failed(t *testing.T) {
	tests := []struct {
		name     string
		input    TargetPodConfig
		expected error
	}{
		{
			name: "empty ContextName field value",
			input: TargetPodConfig{
				ContextName:   "",
				Namespace:     "sample",
				LabelKey:      "run",
				LabelValue:    "sample-api",
				ContainerName: "sample-api",
			},
			expected: fmt.Errorf("targetPod field contextName is required"),
		},
		{
			name: "empty Namespace field value",
			input: TargetPodConfig{
				ContextName:   "gke_sample_asia-east1_gke_sample_asia",
				Namespace:     "",
				LabelKey:      "run",
				LabelValue:    "sample-api",
				ContainerName: "sample-api",
			},
			expected: fmt.Errorf("targetPod field namespace is required"),
		},
		{
			name: "empty LabelKey field value",
			input: TargetPodConfig{
				ContextName:   "gke_sample_asia-east1_gke_sample_asia",
				Namespace:     "sample",
				LabelKey:      "",
				LabelValue:    "sample-api",
				ContainerName: "sample-api",
			},
			expected: fmt.Errorf("targetPod field podLabelKey is required"),
		},
		{
			name: "empty LabelValue filed value",
			input: TargetPodConfig{
				ContextName:   "gke_sample_asia-east1_gke_sample_asia",
				Namespace:     "sample",
				LabelKey:      "run",
				LabelValue:    "",
				ContainerName: "sample-api",
			},
			expected: fmt.Errorf("targetPod field podLabelValue is required"),
		},
		{
			name: "empty ContainerName filed value",
			input: TargetPodConfig{
				ContextName:   "gke_sample_asia-east1_gke_sample_asia",
				Namespace:     "sample",
				LabelKey:      "run",
				LabelValue:    "sample-api",
				ContainerName: "",
			},
			expected: fmt.Errorf("targetPod field containerName is required"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateGetTargetPodRequiredField(tt.input)
			assert.Equal(t, tt.expected, err)
		})
	}
}
