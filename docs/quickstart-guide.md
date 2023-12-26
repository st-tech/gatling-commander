# Quick Start Guide
日本語版Quick Start Guideは[こちら](./quickstart-guide.jp.md)

- [Quick Start Guide](#quick-start-guide)
  - [Create execution environment](#create-execution-environment)
    - [Install CLI module](#install-cli-module)
    - [Install required tools](#install-required-tools)
    - [Create Gatling Operator execution environment](#create-gatling-operator-execution-environment)
    - [Create Google Sheets](#create-google-sheets)
    - [Create configuration file for load test](#create-configuration-file-for-load-test)
    - [Create Kubernetes Manifest of Gatling Resource](#create-kubernetes-manifest-of-gatling-resource)
    - [Authentication for Google Sheets](#authentication-for-google-sheets)
  - [Run load test](#run-load-test)
  - [Record load test results](#record-load-test-results)
  - [Interruput running load test](#interruput-running-load-test)
  - [Notify load test finish](#notify-load-test-finish)
  - [Discontinuation of load test execution due to threshold value](#discontinuation-of-load-test-execution-due-to-threshold-value)
    - [Discontinuation when load test failed](#discontinuation-when-load-test-failed)
    - [Discontinuation when target latency is exceeded](#discontinuation-when-target-latency-is-exceeded)

This describes the minimal information abount how to create the execution environment and and run CLI quickly. More information about configuration files, privileges and authentication, please refer to [User Guide](./user-guide.md).

## Create execution environment
### Install CLI module
You can install zozo-mlops-loadtest-cli with the following command.
```bash
go install github.com/st-tech/gatling-commander@latest
```
### Install required tools
- [Gatling Operator](https://github.com/st-tech/gatling-operator/tree/main)
- [Docker](https://www.docker.com/)
- [Go](https://go.dev/)
  - version: 1.20
- [Google Sheets](https://www.google.com/intl/ja_jp/sheets/about/)
  - Google Sheets is required for recording load test results
- [Google Cloud Project](https://cloud.google.com/resource-manager/docs/creating-managing-projects)
  - Google Cloud Project is required for accessing Google Sheets by [Google Sheets API](https://developers.google.com/sheets/api/guides/concepts)

### Create Gatling Operator execution environment
zozo-mlops-loadtest-cli is intended for use in load test with the [Gatling Operator](https://github.com/st-tech/gatling-operator).  
When using zozo-mlops-loadtest-cli, please create an environment in which the Gatling Operator can be used first. Information about how to setup Gatling Operator environment, please refer to the Gatling Operator [Quick Start Guide](https://github.com/st-tech/gatling-operator/blob/main/docs/quickstart-guide.md).

Please create `gatling` directory in directory which run zozo-mlops-loadtest-cli command, and create required file for Gatling Operator execution. For more information about this directory, please refer to [What is this `gatling` directory?](../gatling/README.md).

### Create Google Sheets
zozo-mlops-loadtest-cli record the load test results to Google Sheets. Both existing and newly created sheet can be used for the recording destination.

Please get Google Sheets ID and grant editor role by doing the following tasks.

- Get Google Sheets ID
  - Open Google Sheets to record the load test results and copy the string corresponding to {ID} in the URL.
    - https://docs.google.com/spreadsheets/d/{ID}/edit#gid=0
  - Set copied string to `services[].spreadsheetID` in `config.yaml`
- Grant role for editing the sheet
  - Please grant Google Sheets editor role for using zozo-mlops-loadtest-cli to the account to be authenticated.
    - Click the Share button in the UI of the sheet you are recording and grant editor role to the target account.

### Create configuration file for load test
Configuration values for the load test are written in `config/config.yaml`.  
Also, in the `base_manifest.yaml` described below in [Create Kubernetes Manifest of Gatling Resource](#create-kubernetes-manifest-of-gatling-resource), fields marked `<config.yaml overrides this field>` will be overwritten by the corresponding value of the field in `config.yaml`.

More information about each field of `config.yaml`, please refer to [User Guide](./user-guide.md).

Here is the example of `config.yaml`.

```yaml
gatlingContextName: gatling-cluster-context-name
imageRepository: gatling-image-stored-repository-url
imagePrefix: gatlinge-image-name-prefix
imageURL: "" # (Optional) specify image url when using pre build gatling container image
baseManifest: config/base_manifest.yaml
gatlingDockerfileDir: gatling
startupTimeoutSec: 1800 # 30min
execTimeoutSec: 10800 # 3h
slackConfig:
  webhookURL: slack-webhook-url
  mentionText: <@targetMemberID>
services:
  - name: sample-service
    spreadsheetID: sample-sheets-id
    failFast: false
    targetPercentile:
    targetLatency:
    targetPodConfig:
      contextName: target-pod-context-name
      namespace: sample-namespace
      labelKey: run
      labelValue: sample-api
      containerName: sample-api
    scenarioSpecs:
      - name: case-1
        subName: 10rps
        testScenarioSpec:
          simulationClass: SampleSimulation
          parallelism: 1
          env:
            - name: ENV
              value: "dev"
            - name: CONCURRENCY
              value: "10"
            - name: DURATION
              value: "180"
      - name: case-2
        subName: 20rps
        testScenarioSpec:
          simulationClass: SampleSimulation
          parallelism: 1
          env:
            - name: ENV
              value: "dev"
            - name: CONCURRENCY
              value: "20"
            - name: DURATION
              value: "180"

```

### Create Kubernetes Manifest of Gatling Resource
zozo-mlops-loadtest-cli creates an object for a Gatling Resource, a Kubernetes Custom Resource used by the Gatling Operator, to run load test.

The `base_manifest.yaml` is Kubernetes manifest for Gatling Resource.  
The `base_manifest.yaml` has the common values for each load test for the Gatling Resource.

Fields marked `<config.yaml overrides this field>` in `base_manifest.yaml` are set to different values for each loadtest. The value of this field will be replaced by the corresponding value in `config.yaml` respectively when zozo-mlops-loadtest-cli is run.  
Therefore, setting values to fields marked `<config.yaml overrides this field>` in `base_manifest.yaml` is not necessary.

For information on how to write Kubernetes manifest for Gatling Resource (`config/base_manifest.yaml`), see the [samples YAML file](https://github.com/st-tech/gatling-operator/blob/main/config/samples/gatling-operator_v1alpha1_gatling01.yaml) which is provided in st-tech/gatling-operator repository, and create manifest for your environment.

For more information, please refer to [User Guide](./user-guide.md) for details.

The following is a sample of `base_manifest.yaml`.

```yaml
apiVersion: gatling-operator.tech.zozo.com/v1alpha1
kind: Gatling
metadata:
  name: <config.yaml overrides this field> # will be overrided by services[].name field value in config.yaml. ex: sample-service
  namespace: gatling
spec:
  generateReport: true
  generateLocalReport: true
  notifyReport: false
  cleanupAfterJobDone: false
  podSpec:
    gatlingImage: <config.yaml overrides this field> # will be overrided by built Gatling Image URL or imageURL field value in config.yaml. ex: asia-docker.pkg.dev/project_id/foo/bar/gatlinge-image-name-prefix-YYYYMMDD
    rcloneImage: rclone/rclone
    resources:
      requests:
        cpu: "7000m"
        memory: "4G"
      limits:
        cpu: "7000m"
        memory: "4G"
    serviceAccountName: "gatling-operator-worker-service-account"
    affinity:
      nodeAffinity:
        requiredDuringSchedulingIgnoredDuringExecution:
          nodeSelectorTerms:
            - matchExpressions:
                - key: cloud.google.com/gke-nodepool
                  operator: In
                  values:
                    - "gatling-operator-worker-pool"
    tolerations:
      - key: "dedicated"
        operator: "Equal"
        value: "gatling-operator-worker-pool"
        effect: "NoSchedule"
  cloudStorageSpec:
    provider: "gcp"
    bucket: "report-storage-bucket-name"
  notificationServiceSpec:
    provider: "slack"
    secretName: "gatling-notification-slack-secrets"
  testScenarioSpec:
    parallelism: <config.yaml overrides this field> # will be overrided by services[].scenarioSpecs[].testScenarioSpec.parallelism field value. ex: 1
    simulationClass: <config.yaml overrides this field> # will be overrided by services[].scenarioSpecs[].testScenarioSpec.simulationClass field value. ex: SampleSimulation
    env: # will be overrided by services[].scenarioSpecs[].testScenarioSpec.env[] field value. ex: `env: [{name: ENV, value: "dev"}, {name: CONCURRENCY, value: "20"}]`
      - name: <config.yaml overrides this field>
        value: <config.yaml overrides this field>

```

### Authentication for Google Sheets
Load test results are recorded in Google Sheets.  
For recording the results, activate [Google Sheets API](https://developers.google.com/sheets/api/guides/concepts) in Google Cloud Project, and authenticate your Google Account or Service Account which has sheet editor role.
```bash
gcloud auth application-default login --scopes=https://www.googleapis.com/auth/cloud-platform,https://www.googleapis.com/auth/spreadsheets
```

## Run load test
The following zozo-mlops-loadtest-cli command will run load test which written in configuration.
```bash
zozo-mlops-loadtest-cli exec --config "config/config.yaml"
```
The `--skip-build` option allows you to skip building a Gatling Image. To use this option, you must set `imageURL` in `config.yaml` to the URL of the Gatling Image you have built.  
If the `--skip-build` option is not specified, a new Gatling Image will always be built.

In `config.yaml`, `services` is an array of configuration values for each service.  
`services[].scenarioSpecs` in `config.yaml` describes an array of configuration values for each load test.

The load tests for each service listed in `services` in `config.yaml` are executed in parallel.  
The load tests listed in `scenarioSpecs` in service are executed in the order in which they are listed.

## Record load test results
The load test results are recorded in Google Sheets specified in `config.yaml`.  
The sheets for recording are created by zozo-mlops-loadtest-cli and are in the format of `services[].name` + `date at runtime` in `config.yaml`. (e.g. `sample-service-20231113`)

If there is load test run with same service name and same date, the results will be recorded to the same sheet. In that case, the results will be appended to the bottom row.

## Interruput running load test
You can interruput the load test run by terminating the running zozo-mlops-loadtest-cli process with `ctrl + c`.  
Upon interruption, the running Gatling object will be deleted immediately.

## Notify load test finish
By specifying the Slack webhook URL in `slackConfig.webhookURL` in `config.yaml`, you can notify Slack when the load test is finished.  
For Slack's Webhook URL, please refer to [Slack API documentation](https://api.slack.com/messaging/webhooks) to get it from the console.

## Discontinuation of load test execution due to threshold value
The load tests specified in `scenarioSpecs` in the service are executed sequentially.  
By setting threshold values in config.yaml, subsequent load tests in the same service can be discontinued according to the results of the Gatling Report after the load test is executed.

### Discontinuation when load test failed
In Gatling load tests, if returned response if different from the one specified in the load test scenario is returned, it is treated as a fail.  
If `failFast` in `config.yaml` is set to `true`, subsequent load tests on the same service will not be performed if the load test results include a failed response.

### Discontinuation when target latency is exceeded
zozo-mlops-loadtest-cli allows you to set a target latency threshold for each service and discontinue subsequent load tests if exceeds the threshold.  
To perform a latency threshold check, set both `targetLatency` and `targetPercentile` in `config.yaml`.

- targetPercentile
  - Specify the percentile value of the threshold. Values can be specified from [50, 75, 95, 99]
- targetLatency
  - Specify latency threshold in milliseconds
