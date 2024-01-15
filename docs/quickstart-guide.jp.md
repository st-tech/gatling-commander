# Gatling Commander クイックスタートガイド
- [Gatling Commander クイックスタートガイド](#gatling-commander-クイックスタートガイド)
  - [事前準備](#事前準備)
    - [モジュールのインストール](#モジュールのインストール)
    - [ツールのインストール](#ツールのインストール)
    - [Gatling Operatorの環境構築](#gatling-operatorの環境構築)
    - [Google Sheetsの作成](#google-sheetsの作成)
    - [負荷試験設定ファイルの作成](#負荷試験設定ファイルの作成)
    - [Gatlingリソースのマニフェスト作成](#gatlingリソースのマニフェスト作成)
    - [Sheets APIへの認証](#sheets-apiへの認証)
  - [負荷試験の実行](#負荷試験の実行)
  - [負荷試験結果の出力](#負荷試験結果の出力)
  - [負荷試験実行の中止](#負荷試験実行の中止)
  - [負荷試験終了の通知](#負荷試験終了の通知)
  - [閾値による負荷試験実行の中止](#閾値による負荷試験実行の中止)
    - [Failした際の中止](#failした際の中止)
    - [目標レイテンシを上回った際の中止](#目標レイテンシを上回った際の中止)

Gatling Commanderをすぐに利用するため、実行環境の作成と実行方法について最小限の情報を記載しています。  
設定ファイル、権限・認証に関する詳しい説明は[User Guide](./user-guide.jp.md)を参照してください。

## 事前準備
### モジュールのインストール
```bash
go install github.com/st-tech/gatling-commander@latest
```
### ツールのインストール
- [Gatling Operator](https://github.com/st-tech/gatling-operator/tree/main)
- [Docker](https://www.docker.com/)
- [Go](https://go.dev/)
  - version: 1.20
- [Google Sheets](https://www.google.com/intl/ja_jp/sheets/about/)
  - 負荷試験結果の書き込み先として事前にシートの作成が必要です
- [Google Cloud Project](https://cloud.google.com/resource-manager/docs/creating-managing-projects)
  - Google Sheetsの認証に必要です

### Gatling Operatorの環境構築
Gatling Commanderは[Gatling Operator](https://github.com/st-tech/gatling-operator)の利用を前提としています。  
[Gatling OperatorのQuick Start Guide](https://github.com/st-tech/gatling-operator/blob/main/docs/quickstart-guide.md)を参考にGatling Operatorを利用可能な環境構築を行なってください。

コマンド実行を行うディレクトリ内に`gatling`ディレクトリを作成し、Gatling Operator実行時に必要なファイルのコピーや作成を行なってください。  
詳細は[What is this `gatling` directory?](../gatling/README.md)を参照してください。

### Google Sheetsの作成
負荷試験結果の記録先として[Google Sheets](https://www.google.com/intl/ja_jp/sheets/about/)を利用しています。  
記録先のシートには、既存のシート・新規作成のシートの両方が利用可能です。

次の作業を実施して記録先のGoogle SheetsのIDの取得と編集者権限の付与をしてください。  

- IDの取得
  - 負荷試験結果の記録を行うGoogle Sheetsを開き、URLの{ID}に該当する文字列をコピーしてください
    - https://docs.google.com/spreadsheets/d/{ID}/edit#gid=0
  - コピーした文字列は`config.yaml`の`services[].spreadsheetID`に設定してください
- シートの権限付与
  - Gatling Commanderを利用する際に、認証するアカウントへGoogle Sheetsの編集者権限を付与してください
    - 記録先のシートのUIから共有ボタンをクリックし、対象のアカウントへ編集者権限を付与できます

### 負荷試験設定ファイルの作成
負荷試験の設定値は`config/config.yaml`に記述します。  
また、[Gatlingリソースのマニフェスト作成](#Gatlingリソースのマニフェスト作成)で後述する`base_manifest.yaml`のうち、`<config.yaml overrides this field>`と記載のあるフィールドは`config.yaml`に記述したフィールドの値により上書きされます。

`config.yaml`の各フィールドに設定する値の詳細は[User Guide](./user-guide.jp.md)を参照してください。

以下は`config.yaml`のサンプルです。

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

### Gatlingリソースのマニフェスト作成
Gatling CommanderではGatling Operatorで利用するKubernetesのCustom ResourceであるGatlingリソースのオブジェクトを作成して負荷試験を行います。

`base_manifest.yaml`はGatlingリソースのKubernetesマニフェストです。  
`base_manifest.yaml`にはGatlingリソースについて、負荷試験ごとに共通の値を記述します。

`base_manifest.yaml`に`<config.yaml overrides this field>`と記載があるフィールドは、負荷試験ごとに異なる値が設定されます。こちらのフィールドの値は、Gatling Commanderの実行時に`config.yaml`の値でそれぞれ置き換えられます。  
そのため`base_manifest.yaml`での値の設定は不要です。

`config/base_manifest.yaml`の記述については、[Gatling Operatorのサンプル](https://github.com/st-tech/gatling-operator/blob/main/config/samples/gatling-operator_v1alpha1_gatling01.yaml)を参考に、利用する環境に合わせて作成してください。  
詳細については[User Guide](./user-guide.jp.md)を参照してください。

以下は`base_manifest.yaml`のサンプルです。


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
    parallelism: 1 # <config.yaml overrides this field> # will be overrided by services[].scenarioSpecs[].testScenarioSpec.parallelism field value. ex: 1
    simulationClass: <config.yaml overrides this field> # will be overrided by services[].scenarioSpecs[].testScenarioSpec.simulationClass field value. ex: SampleSimulation
    env: # will be overrided by services[].scenarioSpecs[].testScenarioSpec.env[] field value. ex: `env: [{name: ENV, value: "dev"}, {name: CONCURRENCY, value: "20"}]`
      - name: <config.yaml overrides this field>
        value: <config.yaml overrides this field>

```

### Sheets APIへの認証
負荷試験結果はGoogle Sheetsに記録されます。  
記録にはシートの編集者権限が必要になるため、Google Cloud Projectで[Google Sheets API](https://developers.google.com/sheets/api/guides/concepts)を有効化し、利用するGoogleアカウント・サービスアカウントを認証してください。
```bash
gcloud auth application-default login --scopes=https://www.googleapis.com/auth/cloud-platform,https://www.googleapis.com/auth/spreadsheets
```

## 負荷試験の実行
次のコマンドにより負荷試験が実行されます。
```bash
gatling-commander exec --config "config/config.yaml"
```
`--skip-build`オプションを指定するとGatling Imageのbuildをスキップできます。  
このオプションを使用するには、`config.yaml`で`imageURL`に予めbuildしたGatling ImageのURLを設定する必要があります。  
`--skip-build`オプションを指定しない場合は、常に新しいGatling Imageがbuildされます。

`config.yaml`の`services`には各serviceごとの設定値を配列で記述します。  
`config.yaml`の`services[].scenarioSpecs`には負荷試験ごとの設定値を配列で記述します。

`config.yaml`の`services`に記載した各serviceごとの負荷試験群は並行で実行されます。  
service内の`scenarioSpecs`に記載した負荷試験は記載順に順次実行されます。

## 負荷試験結果の出力
負荷試験結果は`config.yaml`で指定したGoogle Sheetsに記録されます。  
記録用のシートはGatling Commanderにより作成され、`config.yaml`の`services[].name` + `実行日`の形式で作成されます。（例：`sample-service-20231113`）

同一のservice名を持ち、同じ日付に実施された負荷試験の記録用シートは同名であるため、既存のシートに追記する形で記録されます。  
追記される結果は一番下の行に追加されます。

## 負荷試験実行の中止
`ctrl + c`で実行中のGatling Commanderのプロセスを終了することで、負荷試験実行を中断することができます。  
中断すると実行中のGatling Objectは直ちに削除されます。

## 負荷試験終了の通知
`config.yaml`の`slackConfig.webhookURL`にSlackのWebhook URLを指定することで、負荷試験が終了した際にSlackに通知できます。  
SlackのWebhook URLについては[Slack APIの公式ドキュメント](https://api.slack.com/messaging/webhooks)を参考にコンソールから取得してください。

## 閾値による負荷試験実行の中止
service内の`scenarioSpecs`に指定した負荷試験は順次実行されます。  
負荷試験実行後にGatling Reportの結果に応じて、同一serviceでの以降の負荷試験を中止できます。

### Failした際の中止
Gatlingの負荷試験では、負荷試験シナリオで指定した以外のレスポンスが返された場合にfailとして扱われます。  
`config.yaml`の`failFast`を`true`に設定すると負荷試験結果にfailedが含まれた場合に、同一serviceでの以降の負荷試験を実施しません。

### 目標レイテンシを上回った際の中止
Gatling Commanderではserviceごとに目標レイテンシの閾値を設定し、閾値を超えた場合に以降の負荷試験を中止できます。  
レイテンシの閾値チェックを行うには、`config.yaml`の`targetLatency`・`targetPercentile`の両方を設定します。

- targetPercentile
  - 閾値のパーセンタイル値を指定してください。値は[50, 75, 95, 99]の中から指定可能です
- targetLatency
  - レイテンシの閾値をミリ秒で指定してください
