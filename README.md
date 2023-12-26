# zozo-mlops-loadtest-cli
日本語版READMEは[こちら](./README.jp.md)
## What is zozo-mlops-loadtest-cli ?
zozo-mlops-loadtest-cli is a CLI tool that automates a series of tasks in the execution of load test using [Gatling Operator](https://github.com/st-tech/gatling-operator).  
Gatling Operator is a Kubernetes Operator for running automated distributed Gatling load test.

## Features
By writing load test scenarios in the configuration file, zozo-mlops-loadtest-cli automatically run load test and record the results.

zozo-mlops-loadtest-cli automates the following tasks.
- Create Gatling objects for each load test
- Build Gatling image
- Stop load test when result latency exceeds a predefined threshold
- Record Gatling Report and target container metrics for each load test
- Check running load test status

In addition, zozo-mlops-loadtest-cli allow to have multiple load test scenarios in the configuration file.

After preparing the configuration file, run the `zozo-mlops-loadtest-cli` command, this will automatically run all load test and record the results to [Google Sheets](https://www.google.com/sheets/about/).  
zozo-mlops-loadtest-cli notify load test finished status to [Slack](https://slack.com) as configured in the configuration file.

Please refer to [User Guide](./docs/user-guide.md) about details of each field in the configuration.

Here is an example of how to fill out the configuration file (`config.yaml`).
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
    targetPercentile: 99 # (%ile)
    targetLatency: 500 # (ms)
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

## Requirements
zozo-mlops-loadtest-cli is intended for use in load test with the Gatling Operator.  
When using zozo-mlops-loadtest-cli, please create an environment in which the Gatling Operator can be used first. Information about how to setup Gatling Operator environment, please refer to the Gatling Operator [Quick Start Guide](https://github.com/st-tech/gatling-operator/blob/main/docs/quickstart-guide.md).

## Quick Start
- [Quick Start Guide](./docs/quickstart-guide.md)

## Documentations
- [User Guide](./docs/user-guide.md)
- [Developer Guide](./docs/developer.md)

## Contributing
Please make a GitHub issue or pull request to help us improve this CLI. We expect contributors to comply with the [Contributor Covenant](https://contributor-covenant.org/).


## License
zozo-mlops-loadtest-cli is available as open source under the terms of the MIT License. For more details, see the [LICENSE](./LICENSE) file.
