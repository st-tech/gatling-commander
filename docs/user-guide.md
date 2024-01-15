# User Guide
日本語版User Guideは[こちら](./user-guide.jp.md)

- [User Guide](#user-guide)
  - [About configuration file](#about-configuration-file)
    - [Configuration of load test (`config.yaml`)](#configuration-of-load-test-configyaml)
      - [Hierarchy of `config.yaml`](#hierarchy-of-configyaml)
      - [Configuration values common to the entire load test](#configuration-values-common-to-the-entire-load-test)
      - [Configuration values for each service](#configuration-values-for-each-service)
      - [Configuration values for each load test scenario](#configuration-values-for-each-load-test-scenario)
    - [Manifest of Gatling Resource](#manifest-of-gatling-resource)
  - [Required Role and Authentication](#required-role-and-authentication)
    - [Roles to pull and push docker images](#roles-to-pull-and-push-docker-images)
    - [Roles to get, create, and delete objects in a Kubernetes cluster](#roles-to-get-create-and-delete-objects-in-a-kubernetes-cluster)
    - [Roles to read from Cloud Storage](#roles-to-read-from-cloud-storage)
    - [Roles to read, write Google Sheets](#roles-to-read-write-google-sheets)
      - [Authentication of Google Sheets API](#authentication-of-google-sheets-api)

This describes the information about how to write configuration files and authentication when using Gatling Commander.  
Please refer to the [Quick Start Guide](./quickstart-guide.md) for information on how to install this tool, set up the [Gatling Operator](https://github.com/st-tech/gatling-operator), and prepare and run the load test scenario.
## About configuration file
Gatling Commander requires the following two types of YAML files as configuration files.  
- config.yaml
- base_manifest.yaml

Configuration values for the load test are written in `config/config.yaml`.

The `base_manifest.yaml` describes the common values for each load test in the Kubernetes manifest of the Gatling Resource.

Fields marked `<config.yaml overrides this field>` in `base_manifest.yaml` are set to different values for each load test. The value of this field will be replaced by the value in `config.yaml` respectively when Gatling Commander runs. Therefore, setting values to fields marked `<config.yaml overrides this field>` in `base_manifest.yaml` is not necessary.

\* Gatling Commander once loads `base_manifest.yaml` value to Gatling struct object before it replaces the value by `config.yaml`. So the type of `base_manifest.yaml` field value must be matched to Gatling struct field one. If type not match, an error like following occur.

```go
json: cannot unmarshal string into Go struct field TestScenarioSpec.spec.testScenarioSpec.parallelism of type int32
```

The location and file name of `config.yaml` and `base_manifest.yaml` can be any value.  
For the `config.yaml` path, specify the value of the `--config` option when executing the command.  
For the `base_manifest.yaml` path, set path value to `baseManifest` in `config.yaml`.  

### Configuration of load test (`config.yaml`)
This describes about each field in `config.yaml`

#### Hierarchy of `config.yaml`
The `config.yaml` has a hierarchical structure.

In Gatling Commander, a service is defined as a group of load test.  
A service has one or more load test scenarios for the same target.  
The results of the load test for the same service are recorded in [Google Sheets](https://www.google.com/sheets/about/) specified by `services[].spreadsheetID` in `config.yaml`.

The individual load test scenario settings are defined in the `testScenarioSpec` in `config.yaml`. This is the same as the value of `testScenarioSpec` of the Gatling Object, which is required for a load test with the Gatling Operator.

The top-level field in `config.yaml` specifies common configuration values for the entire load test, and the `services` in `config.yaml` specify configuration values for each service.  
Also, the `services[].testScenarioSpec` in `config.yaml` specifies the setting values for each load test.

Thus, `config.yaml` consists of a nested hierarchical structure of configuration values: `configuration values common to the entire load test -> configuration values for each service -> scenario for each load test`.

#### Configuration values common to the entire load test
This section describes the configuration values in the `config.yaml` that are common to the entire load test.

| Field | Description |
| --- | --- |
| `gatlingContextName` _string_ | (Required) Context name of Kubernetes cluster which Gatling Pod running in.  |
| `imageRepository` _string_ | (Required) Container image repository url in which Gatling image is stored. |
| `imagePrefix` _string_ | (Required) String which is used to add built Gatling image name prefix. |
| `imageURL` _string_ | (Optional) Container image URL. When you run `exec` subcommand with `--skip-build` arguments, you must fill this field to specify Gatling image. |
| `baseManifest` _string_ | (Required) Path of Gatling Kubernetes manifest.  |
| `gatlingDockerfileDir` _string_ | (Required) Path of directory in which Dockerfile for Gatling image is stored. |
| `startupTimeoutSec` _integer_ | (Required) Timeout seconds threshold about each Gatling Job startup. |
| `execTimeoutSec` _integer_ | (Required) Timeout seconds threshold about each Gatling Job running. |
| `slackConfig.webhookURL` _string_ | (Optional) Slack webhook url for notification. If set this value, finished CLI will be notified.  |
| `slackConfig.mentionText` _string_ | (Optional) Slack mention target. If set member_id to this field, CLI notification mention user who has the member_id. The webhookURL field must be specified with this field value. |
| `services` _[]object_ | (Required) This field has some services setting values. |

#### Configuration values for each service
This section describes the configuration values for each service in `config.yaml`.

| Field | Description |
| --- | --- |
| `name` _string_ | (Required) Service name. Please specify any value. Used in Gatling object metadata name and so on.  |
| `spreadsheetID` _string_ | (Required) Google Sheets ID to which load test result will be written. |
| `failFast` _boolean_ | (Required) The flag determining whether to start next load test or not when current load test result failed item count exceeds 0. |
| `targetPercentile` _integer_ | (Optional) Threshold of latency percentile, specify this field value from [50, 75, 95, 99]. If this field value is set, CLI check current load test result specified percentile value and decide whether to start next load test or not. The targetLatency field must be specified with this field value. |
| `targetLatency` _integer_ | (Optional) Threshold of latency milliseconds, this field must be specified with targetPercentile.  |
| `targetPodConfig.contextName` _string_ | (Required) Context name of Kubernetes cluster in which loadtest target Pod running. |
| `targetPodConfig.namespace` _string_ | (Required) Kubernetes namespace in which load test target Pod is running. |
| `targetPodConfig.labelKey` _string_ | (Required) Metadata Labels key of load test target Pod.  |
| `targetPodConfig.labelValue` _string_ | (Required) Metadata Labels value of load test target Pod. |
| `targetPodConfig.containerName` _string_ | (Required) Name of load test target container name which is running in load test target Pod. |
| `scenarioSpecs` _[]object_ | (Required) This field has some scenarioSpecs setting values. |

#### Configuration values for each load test scenario
This section describes the configuration values in `config.yaml` for each individual load test scenario.

| Field | Description |
| --- | --- |
| `name` _string_ | (Required) Load test name which is used as Google Sheets name and so on. |
| `subName` _string_ | (Required) Load test sub name which is used in load test result row subName column. |
| `testScenarioSpec` _object_ | (Required) Gatling object testScenarioSpec field. Please refer gatling-operator document [TestScenarioSpec](https://github.com/st-tech/gatling-operator/blob/main/docs/api.md#testscenariospec). |

### Manifest of Gatling Resource
The `base_manifest.yaml` describes the fields in the Kubernetes manifest of the Gatling Resource that set common values for each load test.  
For more information about the fields in the Kubernetes manifest of the Gatling Resource, see [Gatling Operator API Reference](https://github.com/st-tech/gatling-operator/blob/main/docs/api.md#gatling).

Fields marked `<config.yaml overrides this field>` in `base_manifest.yaml` are set to different values for each loadtest. The value of this field will be replaced by the corresponding value in `config.yaml` respectively when Gatling Commander is run.  
Therefore, setting values to fields marked `<config.yaml overrides this field>` in `base_manifest.yaml` is not necessary.

This section describes the fields in `base_manifest.yaml` that are replaced by values in `config.yaml`.

| Field | Description |
| --- | --- |
| `metadata.name` _string_ | Overwritten by service name loaded from `services[].name` field value in `config.yaml` |
| `spec.podSpec.gatlingImage` _string_ | Overwritten by built Gatling image URL or image URL loaded from `imageURL` field value in `config.yaml` |
| `spec.testScenarioSpec.parallelism` _interger_ | Overwritten by `services[].scenarioSpecs[].testScenarioSpec.parallelism` field value in `config.yaml` |
| `spec.testScenarioSpec.simulationClass` _string_ | Overwritten by `services[].scenarioSpecs[].testScenarioSpec.simulationClass` field value in `config.yaml` |
| `spec.testScenarioSpec.env[]` _[]dict_ | Overwritten by `services[].scenarioSpecs[].testScenarioSpec.env[]` field value in `config.yaml` |

## Required Role and Authentication
The following roles are required to run Gatling Commander.

- Roles to pull and push docker images
- Roles to get, create, and delete objects in a Kubernetes cluster
- Roles to read from Cloud Storage
- Roles to read, write Google Sheets

### Roles to pull and push docker images
If you do not specify `imageURL` in `config.yaml`, it will build a new Gatling Image and push it to the specified Image Repository.  
Gatling Commander currently supports use with Google Cloud only, [Google Artifact Registry](https://cloud.google.com/artifact-registry) and [Google Container Registry](https://cloud.google.com/container-registry/docs/overview) are available.

For building and pushing Gatling Image, please grant the account that is necessary roles to push the Image to an account that is used in the Gatling Commander execution environment.

### Roles to get, create, and delete objects in a Kubernetes cluster
Gatling Commander creates, gets, and deletes Gatling Objects on the specified cluster and fetches metrics for the pods under load test.

Gatling Commander obtains Kubernetes authentication by referring to `$HOME/.kube/config`.  
The account used in the execution environment of Gatling Commander must be authorized to get, create and delete Kubernetes objects.

### Roles to read from Cloud Storage
Gatling Operator creates and uploads a Gatling Report to a `provider` `bucket` specified place which are set in the `cloudStorageSpec` of Gatling manifest.  
Gatling Commander gets the Gatling Report uploaded to the configured `bucket`, reads the target items, and records them in Google Sheets.

Gatling Commander currently supports use with Google Cloud only and reads load test results from Gatling Reports uploaded to Google Cloud Storage.

Please grant the necessary roles to get the Gatling Reports file to the account that is used in the execution environment of Gatling Commander.

### Roles to read, write Google Sheets
Gatling Commander records the load test results in the specified Google Sheets.  
Please grant the editor privilege of the target Google Sheets to the account used in the execution environment of Gatling Commander.

#### Authentication of Google Sheets API
Gatling Commander use [Google Sheets API](https://developers.google.com/sheets/api/guides/concepts) for manipulating Google Sheets. If you do not have a Google Cloud Project, create one and activate the Google Sheets API.  
After creating a Google Sheets sheet, grant the role to edit the sheet to the account used in the execution environment of Gatling Commander.

Please execute the following command to authenticate Google Sheets.
```bash
gcloud auth application-default login --scopes=https://www.googleapis.com/auth/cloud-platform,https://www.googleapis.com/auth/spreadsheets
```
