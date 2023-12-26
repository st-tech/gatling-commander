# zozo-mlops-loadtest-cliの開発方法
- [zozo-mlops-loadtest-cliの開発方法](#zozo-mlops-loadtest-cliの開発方法)
  - [ローカルでの実行](#ローカルでの実行)
  - [テストの実行](#テストの実行)
  - [lint・コードフォーマットの実行](#lintコードフォーマットの実行)
    - [コードフォーマット](#コードフォーマット)
    - [lint](#lint)
      - [ローカル環境での実行](#ローカル環境での実行)
      - [CIでの実行](#ciでの実行)
  - [パッケージの追加方法](#パッケージの追加方法)
  - [GoDoc](#godoc)
  - [CI](#ci)
  - [gatlingディレクトリについて](#gatlingディレクトリについて)

開発時に必要な情報について記載します。

## ローカルでの実行
ローカル環境での実行には、プロジェクトルートディレクトリ配下で次のコマンドを実行します。
```
go run main.go exec --config "config/config.yaml"
```
上記コマンドを動作させるにあたり、事前準備が必要になります。  
事前準備については[Quick Start Guide](./quickstart-guide.jp.md)を参照してください。

## テストの実行
プロジェクトルートディレクトリで次のコマンドを実行することで単体テストが実行できます。  
テスト時に無限ループとなることを避けるため、コマンド実行時に引数でタイムアウト時間を指定することを推奨します。

```
go test -v ./... -timeout 120s
```

## lint・コードフォーマットの実行
### コードフォーマット
プロジェクトルートディレクトリで次のコマンドを実行することでコードがフォーマットされます。

```
go fmt ./...
```

### lint
lintには[golangci-lint](https://github.com/golangci/golangci-lint)を利用しています。  
Macであれば次のようにbrewでインストールできます。
```
brew install golangci-lint
```
その他のインストール方法は[公式ドキュメント](https://golangci-lint.run/usage/install/)を参照ください。

#### ローカル環境での実行
```
golangci-lint run ./...
```

#### CIでの実行
Pull RequestのPush時にCIでlintのチェックが走ります。  
golangci-lintにより問題のある箇所にコメントが付けられます。内容を確認して修正してください。

## パッケージの追加方法
パッケージ追加の際はimport文を追加し、次のコマンドを実行することで依存関係の解決とインストールがされます。  
パッケージ管理にはgo.modファイルとgo.sumファイルを利用しています。どちらも次のコマンドで更新されるため、手動で編集することはありません。

```
go mod tidy
```

Go自体のバージョンを更新する際は、新しいバージョンのGoを環境にインストールし、次のコマンドでgo.modの更新を行なってください。

```
go mod tidy -go=${VERSION}
```

## GoDoc
次のコマンドによりGoDocでパッケージのドキュメントを提供するWebサーバーが起動します。

```bash
# 事前準備
ln -s $(pwd) ${GOROOT}/src

# ドキュメントの表示
go run golang.org/x/tools/cmd/godoc -http=:6060
```

コマンド実行後に`localhost:6060`にアクセスすることで、モジュールで利用しているパッケージのドキュメントを閲覧できます。

デフォルトではexportされている関数、変数のドキュメントのみ表示されます。  
全ての関数、変数のドキュメントを表示するには`localhost:6060?m=all`へアクセスします。

## CI
Pull RequestのPush時に`main.yaml`に記載のワークフローがトリガされます。  
CIでは次の項目のチェックを行なっています。
- lint
- test

## gatlingディレクトリについて
`gatling`ディレクトリはGatling Operatorの実行に必要なファイルを含むディレクトリです。  
更新する際は[st-tech/gatling-operator/gatling](https://github.com/st-tech/gatling-operator/tree/main/gatling)をコピーしてください。  
詳細は[What is this `gatling` directory?](../gatling/README.md)を参照してください。
