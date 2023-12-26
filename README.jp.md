# zozo-mlops-loadtest-cli
## zozo-mlops-loadtest-cliとは？
zozo-mlops-loadtest-cliは、[Gatling Operator](https://github.com/st-tech/gatling-operator)を使用した負荷試験実施における一連の作業を自動化するCLIツールです。  
Gatling Operatorとは、オープンソースの負荷試験ツールである[Gatling](https://gatling.io/)を利用して、自動分散負荷試験を行うためのKubernetes Operatorです。
## 特徴
負荷試験シナリオを設定ファイルに記述すれば、自動的に負荷試験を実施し結果を記録することができます。

zozo-mlops-loadtest-cliにより次の作業が自動化されます。
- 負荷試験ごとのシナリオに応じたGatlingオブジェクトの作成
- Gatling Imageのビルド
- 過負荷時の負荷試験自動停止
- 負荷試験ごとにGatling Report、コンテナメトリクスを記録
- 実行中の負荷試験の実施状況確認

またzozo-mlops-loadtest-cliでは、設定ファイルに複数の負荷試験シナリオを記述可能です。

設定ファイルの作成後に、zozo-mlops-loadtest-cliのコマンドを実行すると、zozo-mlops-loadtest-cliは全ての負荷試験を実施し、結果を[Google Sheets](https://www.google.com/sheets/about/)に書き込みます。  
また、負荷試験の完了ステータスを[Slack](https://slack.com)通知するように設定することも可能です。

設定ファイルの各フィールドの説明は[User Guide](./docs/user-guide.jp.md)に記載しています。

以下は設定ファイル（`config/config.yaml`）の記入例です。  

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

## 必須条件
zozo-mlops-loadtest-cliはGatling Operatorを使った負荷試験での利用を前提としています。  
利用時はまず、[Gatling OperatorのQuick Start Guide](https://github.com/st-tech/gatling-operator/blob/main/docs/quickstart-guide.md)を参考にGatling Operatorを利用可能な環境を構築してください。

## Google Cloud以外の環境での利用
Gatling Operatorがサポートしている実行環境のうち、zozo-mlops-loadtest-cliでは現状[Google Cloud](https://cloud.google.com/)での利用のみサポートしています。

## クイックスタート
- [Quick Start Guide](./docs/quickstart-guide.jp.md)

## ドキュメント
- [User Guide](./docs/user-guide.jp.md)
- [Developer Guide](./docs/developer.jp.md)

## Contributing
IssueやPull Requestの作成など、コントリビューションは誰でも歓迎です。コントリビューターは[Contributor Covenant](https://contributor-covenant.org/)を遵守することを期待します。

## License
zozo-mlops-loadtest-cliはMITライセンスを適応してオープンソースとして公開しています。[LICENSE](./LICENSE) を参照してください。
