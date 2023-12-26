# How to develop Gatling Commander
日本語版Developer Guideは[こちら](./developer.jp.md)

- [How to develop Gatling Commander](#how-to-develop-gatling-commander)
  - [Run command when developing](#run-command-when-developing)
  - [Run unit test](#run-unit-test)
  - [Run lint and code format](#run-lint-and-code-format)
    - [Code format](#code-format)
    - [lint](#lint)
      - [Run lint in local environment](#run-lint-in-local-environment)
      - [Check lint in CI](#check-lint-in-ci)
  - [Add package](#add-package)
  - [GoDoc](#godoc)
  - [CI](#ci)
  - [Abount gatling directory](#abount-gatling-directory)

This describes the information about how to develop Gatling Commander.

## Run command when developing
In your local environment, you can run with this command in project root directory.
```bash
go run main.go exec --config "config/config.yaml"
```
By running this command, some preparation is needed. Information about how to preparate for run Gatling Commander, please refer to [Quick Start Guide](./quickstart-guide.md).

## Run unit test
By running this command in project root directory, all unit tests will be run.  
It is recommended to set timeout with command arguments to avoid infinite loop when running unit tests.

```bash
go test -v ./... -timeout 120s
```

## Run lint and code format
### Code format
By running this command in project root directory, code will be formatted.

```bash
go fmt ./...
```

### lint
We use [golangci-lint](https://github.com/golangci/golangci-lint) for lint.  
If you are on a Mac, you can install it with brew as follows.

```bash
brew install golangci-lint
```
About other way, please refer to golangci-lint [installation document](https://golangci-lint.run/usage/install/).

#### Run lint in local environment
```bash
golangci-lint run ./...
```

#### Check lint in CI
When pushing a Pull Request to head branch, GitHub Actions workflow will be run and check lint with golangci-lint.  
If something goes wrong during the lint check, golangci-lint will add a comment to the line abount what failed to lint. Please check the details and correct the commented line.

## Add package
When adding Go packages, add an import statement and execute the following command to resolve dependencies and install the packages.  
The `go.mod` and `go.sum` files are used to manage dependencies. Both are updated by the following command, so you don't need to update them by yourself.

```bash
go mod tidy
```

To update the Go version, please install a newer version of Go and update the `go.mod` file with the following command.

```bash
go mod tidy -go=${VERSION}
```

## GoDoc
To read package documents with GoDoc, please run the following command and run webserver.

```bash
# preparation
ln -s $(pwd) ${GOROOT}/src

# run webserver
go run golang.org/x/tools/cmd/godoc -http=:6060
```

After the webserver is started, the documentation for the package used by Gatling Commander is displayed with access to `localhost:6060`.

By default, only documents of exported functions and variables are displayed. To see documents of all functions and variables, access to `localhost:6060?m=all`.

## CI
We use GitHub Actions as CI. The workflow described in `main.yaml` is triggered when a Pull Request is pushed.  
This workflow checks the following items.
- lint
- test

## Abount gatling directory
The `gatling` directory has some files which is necessary to run Gatling Operator.  
Please copy [st-tech/gatling-operator/gatling](https://github.com/st-tech/gatling-operator/tree/main/gatling) when you update. For more information about this directory, please refer to [What is this `gatling` directory?](../gatling/README.md).
