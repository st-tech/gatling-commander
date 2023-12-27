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

import gatlingv1alpha1 "github.com/st-tech/gatling-operator/api/v1alpha1"

// Service has common field among each loadtests per target service, and has several its ScenarioSpecs.
type Service struct {
	Name             string          `yaml:"name"`
	SpreadsheetId    string          `yaml:"spreadsheetID"`
	FailFast         bool            `yaml:"failFast"`
	TargetPodConfig  TargetPodConfig `yaml:"targetPodConfig"`
	TargetPercentile uint32          `yaml:"targetPercentile"`
	TargetLatency    float64         `yaml:"targetLatency"`
	ScenarioSpecs    []ScenarioSpec  `yaml:"scenarioSpecs"`
}

// SlackConfig has field which used for slack alert.
type SlackConfig struct {
	WebhookURL  string `yaml:"webhookURL"`
	MentionText string `yaml:"mentionText"`
}

// ScenarioSpec has each loadtest setting field.
type ScenarioSpec struct {
	Name             string                           `yaml:"name"`
	SubName          string                           `yaml:"subName"`
	TestScenarioSpec gatlingv1alpha1.TestScenarioSpec `yaml:"testScenarioSpec"`
}

// TargetPodConfig field value is used to fetch target container metrics value.
type TargetPodConfig struct {
	ContextName   string `yaml:"contextName"`
	Namespace     string `yaml:"namespace"`
	LabelKey      string `yaml:"labelKey"`
	LabelValue    string `yaml:"labelValue"`
	ContainerName string `yaml:"containerName"`
}
