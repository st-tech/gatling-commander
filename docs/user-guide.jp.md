# Gatling Commander ユーザーガイド
- [Gatling Commander ユーザーガイド](#gatling-commander-ユーザーガイド)
  - [設定ファイルの概要](#設定ファイルの概要)
    - [負荷試験設定](#負荷試験設定)
      - [`config.yaml`の階層](#configyamlの階層)
      - [負荷試験全体の設定値](#負荷試験全体の設定値)
      - [serviceの設定値](#serviceの設定値)
      - [負荷試験シナリオの設定値](#負荷試験シナリオの設定値)
    - [Gatling リソースのマニフェスト](#gatling-リソースのマニフェスト)
  - [権限と認証](#権限と認証)
    - [docker imageをpull・pushできる権限](#docker-imageをpullpushできる権限)
    - [Kubernetesクラスタでオブジェクトを読み取り・書き込み・削除できる権限](#kubernetesクラスタでオブジェクトを読み取り書き込み削除できる権限)
    - [Cloud Storageからの読み取り権限](#cloud-storageからの読み取り権限)
    - [Google Sheetsへの読み取り・書き込み権限](#google-sheetsへの読み取り書き込み権限)
      - [Google Sheets APIの認証](#google-sheets-apiの認証)

Gatling Commanderを利用する際の設定ファイルの書き方や認証について説明しています。  
ツールのインストールや[Gatling Operator](https://github.com/st-tech/gatling-operator)のセットアップ、負荷試験シナリオの作成などの事前準備・実行方法については[Quick Start Guide](./quickstart-guide.jp.md)を参照してください。
## 設定ファイルの概要
Gatling Commanderでは、設定ファイルとして次の2種類のYAMLファイルを用意する必要があります。  
- config.yaml
- base_manifest.yaml

負荷試験の設定値は`config.yaml`に記載します。

`base_manifest.yaml`にはGatlingリソースのKubernetesマニフェストのうち、負荷試験ごとに共通の値を記述します。

`base_manifest.yaml`に`<config.yaml overrides this field>`と記載があるフィールドは、負荷試験ごとに異なる値が設定されます。こちらのフィールドの値は、Gatling Commanderの実行時に`config.yaml`の値でそれぞれ置き換えられます。  
そのため`base_manifest.yaml`での値の設定は不要です。

`config.yaml`・`base_manifest.yaml`の保存場所、ファイル名は任意の値を指定可能です。  
参照する`config.yaml`のパスについては、コマンド実行時に`--config`オプションの値を指定してください。  
参照する`base_manifest.yaml`のパスについては、`config.yaml`の`baseManifest`に設定してください。  

### 負荷試験設定
`config.yaml`の各フィールドについて説明します。

#### `config.yaml`の階層
`config.yaml`は階層構造となっています。

Gatling Commanderでは、個々の負荷試験のグループとしてserviceを定義します。  
serviceは同一の負荷試験対象に関する1つ以上の負荷試験シナリオを持ちます。  
同一serviceの負荷試験の結果は、`config.yaml`の`services[].spreadsheetID`で指定した[Google Sheets](https://www.google.com/sheets/about/)に記録されます。

個々の負荷試験シナリオ設定は`config.yaml`の`testScenarioSpec`に定義します。これはGatling Operatorのみを利用して負荷試験を行う場合に設定するGatling Objectの`testScenarioSpec`の値と同じです。

`config.yaml`のトップレベルのフィールドでは負荷試験全体で共通の設定値を指定し、`config.yaml`の`services`には各serviceごとの設定値を指定します。  
また`config.yaml`の`services[].testScenarioSpec`には負荷試験ごとの設定値を指定します。

このように`config.yaml`では、`負荷試験全体に共通の設定値 -> serviceごとの設定値 -> 負荷試験ごとのシナリオ`と設定値がネストされた階層構造で構成されています。

#### 負荷試験全体の設定値
`config.yaml`のうち、負荷試験全体で共通の設定値について説明します。

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

#### serviceの設定値
`config.yaml`のうち、serviceごとの設定値について説明します。

| Field | Description |
| --- | --- |
| `name` _string_ | (Required) Service name. Please specify any value. Used in Gatling object metadata name and so on.  |
| `spreadsheetID` _string_ | (Required) Google Sheets ID to which load test result will be written. |
| `failFast` _boolean_ | (Required) The flag determining whether start next load test or not when current load test result failed item value count exceeds 0. |
| `targetPercentile` _integer_ | (Optional) Threshold of latency percentile, specify this field value from [50, 75, 95, 99]. If this field value is set, CLI check current load test result specified percentile value and whether decide to start next load test or not. The targetLatency field must be specified with this field value. |
| `targetLatency` _integer_ | (Optional) Threshold of latency milliseconds, this field must be specified with targetPercentile.  |
| `targetPodConfig.contextName` _string_ | (Required) Context name of Kubernetes cluster which loadtest target Pod running in. |
| `targetPodConfig.namespace` _string_ | (Required) Kubernetes namespace in which load test target Pod is running. |
| `targetPodConfig.labelKey` _string_ | (Required) Metadata Labels key of load test target Pod.  |
| `targetPodConfig.labelValue` _string_ | (Required) Metadata Labels value of load test target Pod. |
| `targetPodConfig.containerName` _string_ | (Required) Name of load test target container name which is running in load test target Pod. |
| `scenarioSpecs` _[]object_ | (Required) This field has some scenarioSpecs setting values. |

#### 負荷試験シナリオの設定値
`config.yaml`のうち、個々の負荷試験シナリオごとの設定値について説明します。

| Field | Description |
| --- | --- |
| `name` _string_ | (Required) Load test name which is used as Google Sheets name and so on. |
| `subName` _string_ | (Required) Load test sub name which is used in load test result row subName column. |
| `testScenarioSpec` _object_ | (Required) Gatling object testScenarioSpec field. Please refer gatling-operator document [TestScenarioSpec](https://github.com/st-tech/gatling-operator/blob/main/docs/api.md#testscenariospec). |

### Gatling リソースのマニフェスト
`base_manifest.yaml`にはGatlingリソースのKubernetesマニフェストのうち、負荷試験ごとに共通する値を設定するフィールドを記述します。  
GatlingリソースのKubernetesマニフェストのフィールドについては、[Gatling OperatorのAPI Reference](https://github.com/st-tech/gatling-operator/blob/main/docs/api.md#gatling)を参照してください。

`base_manifest.yaml`に`<config.yaml overrides this field>`と記載があるフィールドは、負荷試験ごとに異なる値が設定されます。これらのフィールドの値は、Gatling Commanderの実行時に`config.yaml`の値でそれぞれ置き換えられます。そのため、`base_manifest.yaml`での値の変更は不要です。

※ Gatling Commanderは`base_manifest.yaml`の値を`config.yaml`の値で置き換える前に、一度GoのGatling構造体のオブジェクトにその値を読み込みます。そのため、`base_manifest.yaml`の各フィールドの値の型はGatling構造体の各フィールドの値の型と一致する必要があります。型が一致しない場合次のようなエラーが発生します。

```go
json: cannot unmarshal string into Go struct field TestScenarioSpec.spec.testScenarioSpec.parallelism of type int32
```


`base_manifest.yaml`のうち、`config.yaml`の値で置き換えられるフィールドについて説明します。

| Field | Description |
| --- | --- |
| `metadata.name` _string_ | Overwritten by service name loaded from `services[].name` field value in `config.yaml` |
| `spec.podSpec.gatlingImage` _string_ | Overwritten by built Gatling image URL or image URL loaded from `imageURL` field value in `config.yaml` |
| `spec.testScenarioSpec.parallelism` _interger_ | Overwritten by `services[].scenarioSpecs[].testScenarioSpec.parallelism` field value in `config.yaml` |
| `spec.testScenarioSpec.simulationClass` _string_ | Overwritten by `services[].scenarioSpecs[].testScenarioSpec.simulationClass` field value in `config.yaml` |
| `spec.testScenarioSpec.env[]` _[]dict_ | Overwritten by `services[].scenarioSpecs[].testScenarioSpec.env[]` field value in `config.yaml` |

## 権限と認証
Gatling Commanderの実行には次の権限が必要です。

- docker imageをpull・pushできる権限
- Kubernetesクラスタでオブジェクトの取得・作成・削除ができる権限
- [Cloud Storage](https://cloud.google.com/storage)からの読み取り権限
- Google Sheetsへの読み取り・書き込み権限

### docker imageをpull・pushできる権限
`config.yaml`の`imageURL`を指定しない場合、新しくGatling Imageをbuildし指定したImage Repositoryにpushします。  
Gatling Commanderでは現状Google Cloudのみでの利用をサポートしており、[Google Artifact Registry](https://cloud.google.com/artifact-registry)・[Google Container Registry](https://cloud.google.com/container-registry/docs/overview)が利用可能です。

Gatling Imageのbuild・pushを行う場合は、Gatling Commanderの実行環境で認証されるアカウントにImageをpushするために必要な権限を付与してください。

### Kubernetesクラスタでオブジェクトを読み取り・書き込み・削除できる権限
Gatling Commanderでは指定したクラスタでGatling Objectの作成・取得・削除や負荷試験対象のPodのメトリクスの取得を行います。

Kubernetesの認証情報は`$HOME/.kube/config`を参照して取得しています。  
Gatling Commanderの実行環境で認証されるアカウントにKubernetesオブジェクトの取得・作成・削除ができる権限を付与してください。

### Cloud Storageからの読み取り権限
Gatling Operatorの仕様として、負荷試験実行後にGatling Reportが`cloudStorageSpec`で設定した`provider`の`bucket`に出力されます。  
Gatling Commanderでは設定した`bucket`にアップロードされたGatling Reportを取得し、対象の項目を読み取ってGoogle Sheetsに記録します。

Gatling Commanderでは現状Google Cloudのみでの利用をサポートしており、Google Cloud StorageにアップロードされたGatling Reportから負荷試験結果の読み取りを行います。

Gatling Commanderの実行環境で認証されるアカウントにファイルを取得するために必要な権限付与してください。

### Google Sheetsへの読み取り・書き込み権限
Gatling Commanderでは負荷試験結果を指定されたGoogle Sheetsに記録します。  
Gatling Commanderの実行環境で認証されるアカウントに、対象のGoogle Sheetsの編集者権限を付与してください。

#### Google Sheets APIの認証
Gatling CommanderでGoogle Sheetsを操作する際には[Google Sheets API](https://developers.google.com/sheets/api/guides/concepts)を利用します。Google Cloud Projectがない場合はProjectを作成し、Google Sheets APIを有効化してください。  
Google Sheetsのシートを作成後、認証するアカウントへシートの編集権限を付与してください。

次のコマンドを実行し、Google Sheetsの認証を行なってください。
```bash
gcloud auth application-default login --scopes=https://www.googleapis.com/auth/cloud-platform,https://www.googleapis.com/auth/spreadsheets
```
