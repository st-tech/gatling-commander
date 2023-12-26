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

// Package exec implements command which exec loadtest specified in config.yaml.
package exec

import (
	"context"
	"errors"
	"fmt"
	"os"
	osExec "os/exec"
	"os/signal"
	"sync"
	"time"

	cfg "github.com/st-tech/gatling-commander/pkg/config"
	"github.com/st-tech/gatling-commander/pkg/external/cloudstorages"
	slackTools "github.com/st-tech/gatling-commander/pkg/external/slack"
	sheetTools "github.com/st-tech/gatling-commander/pkg/external/spreadsheet"
	gatlingTools "github.com/st-tech/gatling-commander/pkg/internal/gatling"
	kubeapiTools "github.com/st-tech/gatling-commander/pkg/internal/kubeapi"

	"github.com/spf13/cobra"
	gatlingv1alpha1 "github.com/st-tech/gatling-operator/api/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

type execFlags struct {
	skipBuild bool
}

type loadtestExecError struct {
	serviceName  string
	scenarioName string
	err          error
}

type metricsUsageRatio struct {
	cpu    float64 // ex: 0.1 (10%)
	memory float64
}

type cloudStorageOperator interface {
	Fetch(ctx context.Context, path string) ([]byte, error)
}

type notifyOperator interface {
	Notify(msg string) error
}

type serviceConfig struct {
	name             string
	spreadsheetId    string
	failFast         bool
	targetLatency    float64
	targetPercentile uint32
}

type checkContinueToExecResult struct {
	shouldContinue bool
	message        string
}

func newExecFlags() *execFlags {
	f := &execFlags{}
	return f
}

func (f *execFlags) addFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&f.skipBuild, "skip-build", false, "skip build flag")
}

func (f *execFlags) validateFlags(config *cfg.Config) error {
	if f.skipBuild && config.ImageURL == "" {
		return fmt.Errorf("skip-build flag specified, but there is no imageURL value")
	}
	return nil
}

// NewCmdExec creates the `exec` command.
func NewCmdExec(baseName string, config *cfg.Config) *cobra.Command {
	flags := newExecFlags()

	// Need setting logger to avoid controller-runtime error
	// error: ([controller-runtime] log.SetLogger(...) was never called, logs will not be displayed:).
	ctrl.SetLogger(zap.New())

	cmd := &cobra.Command{
		Use:   "exec",
		Short: "Load configuration, execute load test, and record result",
		Long: `The exec command load configuration file which has specified path with config arguments.
		And execute load test by creating Gatling Resource in the cluster.
		This command load Gatling Report and get load test target container metrics, and record it in specified Google Sheets.
		Complete documentation is available at https://github.com/st-tech/gatling-commander/docs`,
		RunE: func(cmd *cobra.Command, args []string) error {
			err := runExec(cmd, config, flags)
			if config.SlackConfig.WebhookURL != "" {
				isSuccess := false
				if err == nil {
					isSuccess = true
				}
				err := runNotify(config.SlackConfig, isSuccess)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error failed to notify slack %v\n", err)
				} else {
					fmt.Printf("notify to slack succeeded\n")
				}
			}
			return err
		},
	}

	flags.addFlags(cmd)
	return cmd
}

/*
runExec execute all loadtests written in config/config.yaml.

Per-service loadtests are run in parallel.
Each loadtest of service run in order.
If failFast or targetLatency value is set and loadtest finished with this condition, next loadtest of same service is
not executed. (checkContinueToExec)
The error occured in each loadtest will be output after all loadtest finished.
*/
func runExec(cmd *cobra.Command, config *cfg.Config, flags *execFlags) error {
	ctx, cancel := context.WithCancel(context.Background())
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt)
	defer func() {
		signal.Stop(signalCh)
		cancel()
	}()
	go func() {
		<-signalCh
		cancel()
	}()

	/*
		Build and push image.
		https://go.dev/src/time/format.go
		Format is YYYYMMDDhhmm.
	*/
	execDate := time.Now().Format("200601021504")
	imgTag := config.ImagePrefix + "-" + execDate
	var imgURL string
	if err := flags.validateFlags(config); err != nil {
		return fmt.Errorf("config param or argument invalid %v", err)
	}
	if !flags.skipBuild {
		genImageURL, err := buildPushImage(config.ImageRepository, imgTag, config.GatlingDockerfileDir)
		if err != nil {
			return fmt.Errorf("gatling image build error %v", err)
		}
		imgURL = genImageURL
	} else {
		imgURL = config.ImageURL
	}

	/*
		Create channel for receive error in each loadtest run (runLoadtestAndRecord).
		Set number of loadtests of service as max error buffer length.
		Each loadtest run returns at most one error, so loadtestErrorCh buffer is less than or equal to service num.
	*/
	loadtestErrorCh := make(chan loadtestExecError, len(config.Services))

	wg := new(sync.WaitGroup)
	for _, service := range config.Services {
		wg.Add(1)
		go func(ctx context.Context, s cfg.Service) {
			defer wg.Done()
			serviceConfig := extractServiceConfig(s)
			for _, scenarioSpec := range s.ScenarioSpecs {
				serviceName := serviceConfig.name
				scenarioName := scenarioSpec.Name
				scenarioSubName := scenarioSpec.SubName
				/*
					loadtestExecError struct type has err field.
					when error occured, write it to this field and log at parent function too.
				*/
				occuredErr := loadtestExecError{
					serviceName:  serviceName,
					scenarioName: fmt.Sprintf("%v %v", scenarioName, scenarioSubName),
				}
				gatlingReport, err := runLoadtestAndRecord(
					ctx,
					config.GatlingContextName,
					imgURL,
					config.BaseManifest,
					config.StartupTimeoutSec,
					config.ExecTimeoutSec,
					serviceConfig,
					s.TargetPodConfig,
					scenarioSpec,
				)
				if err != nil {
					occuredErr.err = err
					loadtestErrorCh <- occuredErr
					return
				}
				checkContinue, err := checkContinueToExec(serviceConfig, *gatlingReport)
				if err != nil {
					occuredErr.err = err
					loadtestErrorCh <- occuredErr
					return
				}
				// This cancel condition is not caused by the loadtest error, so only log and finish goroutine.
				if !checkContinue.shouldContinue {
					fmt.Printf(
						"service %v loadtest %v execution canceled, %v\n",
						occuredErr.serviceName,
						occuredErr.scenarioName,
						checkContinue.message,
					)
					return
				}
			}
		}(ctx, service)
	}
	wg.Wait()
	close(loadtestErrorCh)
	if len(loadtestErrorCh) > 0 {
		for result := range loadtestErrorCh {
			fmt.Fprintf(
				os.Stderr,
				"Error: failed to run loadtest service %v scenario %v, error %v\n",
				result.serviceName,
				result.scenarioName,
				result.err,
			)
		}
		return fmt.Errorf("more than one loadtest scenario failed")
	}
	return nil
}

/*
runLoadtestAndRecord is main logic in exec command.

runLoadtestAndRecord Create gatling object and run loadtest, fetch loadtest target container metrics.
Wait loadtest running and get gatling report, write report to spreadsheet.
*/
func runLoadtestAndRecord(
	ctx context.Context,
	k8sCtxName string,
	imgURL, manifestPath string,
	waitStartupTimeout int32,
	waitExecTimeout int32,
	serviceConfig serviceConfig,
	targetPodConfig cfg.TargetPodConfig,
	scenarioSpec cfg.ScenarioSpec,
) (*gatlingTools.GatlingReport, error) {
	scenarioName := scenarioSpec.Name
	serviceName := serviceConfig.name

	fmt.Printf("Start service %v loadtest %v\n", serviceName, scenarioName)

	gatling, err := loadAndPatchBaseGatling(serviceName, imgURL, scenarioSpec, manifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to patch gatling struct field, %v", err)
	}

	k8sTargetPodClint, err := kubeapiTools.InitClient(targetPodConfig.ContextName)
	fmt.Printf("service %v loadtest %v, k8s target pod client initialized\n", serviceName, scenarioName)
	if err != nil {
		return nil, fmt.Errorf("failed to init target pod k8s cluster client, %v", err)
	}

	// Fetch resources limit value before run loadtest.
	containerResourcesLimit, err := kubeapiTools.FetchContainerResourcesLimit(ctx, k8sTargetPodClint, targetPodConfig)
	fmt.Printf("service %v loadtest %v, target pod resources limit fetched\n", serviceName, scenarioName)
	if err != nil {
		return nil, fmt.Errorf("failed to get pod spec resources %v", err)
	}

	k8sGatlingClient, err := kubeapiTools.InitClient(k8sCtxName)
	fmt.Printf("service %v loadtest %v, k8s gatling client initialized\n", serviceName, scenarioName)
	if err != nil {
		return nil, fmt.Errorf("failed to init k8s cluster client, %v", err)
	}

	err = gatlingTools.CreateGatling(ctx, k8sGatlingClient, gatling)
	fmt.Printf("service %v loadtest %v, Gatling Object created\n", serviceName, scenarioName)
	if err != nil {
		return nil, fmt.Errorf("failed to create gatling object, %v", err)
	}

	err = gatlingTools.WaitGatlingJobStartup(ctx, k8sGatlingClient, gatling, waitStartupTimeout)
	fmt.Printf("service %v loadtest %v, Gatling Job Started\n", serviceName, scenarioName)
	if err != nil {
		return nil, fmt.Errorf("failed to wait gatling job start, %v", err)
	}

	metricsCl, err := kubeapiTools.InitMetricsClient(targetPodConfig.ContextName)
	fmt.Printf("service %v loadtest %v, k8s target pod metrics client initialized\n", serviceName, scenarioName)
	if err != nil {
		return nil, fmt.Errorf("failed to init k8s client for fetch metrics")
	}

	wg := new(sync.WaitGroup)
	wg.Add(1)
	informJobFinishCh := make(chan bool, 1)
	metricsUsageCh := make(chan kubeapiTools.MetricsField, 1)
	// Fetch target container metrics in background during loadtest running.
	go kubeapiTools.FetchContainerMetricsMean(ctx, wg, metricsCl, metricsUsageCh, informJobFinishCh, targetPodConfig)

	// Wait until Gatling job completed.
	fmt.Printf("service %v loadtest %v, waiting Gatling Job Running\n", serviceName, scenarioName)
	err = gatlingTools.WaitGatlingJobRunning(ctx, k8sGatlingClient, gatling, waitExecTimeout, informJobFinishCh)
	fmt.Printf("service %v loadtest %v, waiting Gatling Job completed\n", serviceName, scenarioName)
	if err != nil {
		return nil, fmt.Errorf("failed to wait gatling job running, %v", err)
	}
	close(informJobFinishCh)

	wg.Wait() // Wait FetchContainerMetricsMean execution finish.
	close(metricsUsageCh)
	metricsUsageMean, ok := <-metricsUsageCh
	if !ok {
		fmt.Fprintf(os.Stderr, "metricsUsageCh value is empty, so each metricsUsage field value is 0")
	}

	// Calculate ratio from usage mean and resource limit.
	metricsUsageRatio := metricsUsageRatio{
		cpu:    kubeapiTools.CalcAndRoundMetricsRatio(metricsUsageMean.Cpu, containerResourcesLimit.Cpu),
		memory: kubeapiTools.CalcAndRoundMetricsRatio(metricsUsageMean.Memory, containerResourcesLimit.Memory),
	}

	storageOp, err := cloudstorages.NewGoogleCloudStorageOperator(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to init cloud storage operator client, %v", err)
	}

	// Fetch storage path from Gatling object and fetch gatlingReport from storage. And parse jsonBytes of gatlingReport
	// to GatlingReport object.
	gatlingReport, err := loadGatlingReportFromCloudStorage(ctx, storageOp, k8sGatlingClient, gatling)
	if err != nil {
		return nil, fmt.Errorf("failed to load gatling report from cloud storage, %v", err)
	}

	// Write loadtest report to spreadsheet.
	fmt.Printf("service %v loadtest %v, start to write Gatling Report to Spreadsheets\n", serviceName, scenarioName)
	err = writeReportToSpreadsheets(ctx, imgURL, serviceConfig, scenarioSpec, gatlingReport, metricsUsageRatio)
	if err != nil {
		return nil, fmt.Errorf("failed to write gatling report to spreadsheets, %v", err)
	}

	fmt.Printf("service %v loadtest %v succeeded\n", serviceName, scenarioName)
	return gatlingReport, nil
}

// runNotify check config.yaml webhookURL parameter and notify loadtest finished to slack.
func runNotify(slackConfig cfg.SlackConfig, isSuccess bool) error {
	webhookURL := slackConfig.WebhookURL
	mention := slackConfig.MentionText

	// skip notify to slack
	if webhookURL == "" {
		fmt.Printf("slack webhook url not found, skip notify to slack\n")
		return nil
	}

	slackOp := slackTools.NewSlackOperator(webhookURL)
	data := slackTools.GenerateSlackPayloadData(mention, isSuccess)
	if err := notifyLoadtestResult(slackOp, data); err != nil {
		return err
	}
	return nil
}

func buildPushImage(imgRepo string, imgTag string, gatlingDockerfileDir string) (string, error) {
	imgURL := fmt.Sprintf("%s:%s", imgRepo, imgTag)
	buildArgs := []string{
		"build",
		"--platform",
		"linux/x86_64",
		"-t",
		imgURL,
		"-f",
		fmt.Sprintf("%s/Dockerfile", gatlingDockerfileDir),
		fmt.Sprintf("./%s", gatlingDockerfileDir),
	}

	buildCmd := osExec.Command("docker", buildArgs...)
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr

	if err := buildCmd.Run(); err != nil {
		return "", err
	}

	authCmd := osExec.Command("gcloud", "auth", "configure-docker")
	authCmd.Stdout = os.Stdout
	authCmd.Stderr = os.Stderr
	if err := authCmd.Run(); err != nil {
		return "", err
	}

	pushCmd := osExec.Command("docker", "push", imgURL)
	pushCmd.Stdout = os.Stdout
	pushCmd.Stderr = os.Stderr
	if err := pushCmd.Run(); err != nil {
		return "", err
	}
	return imgURL, nil
}

/*
loadAndPatchBaseGatling load k8s gatling manifest to gatling object and set config.yaml value to replace target field
in base_manifest.yaml.
*/
func loadAndPatchBaseGatling(
	serviceName string,
	imgURL string,
	scenarioSpec cfg.ScenarioSpec,
	baseManifest string,
) (*gatlingv1alpha1.Gatling, error) {
	gatling, err := gatlingTools.LoadGatlingManifest(baseManifest)
	if err != nil {
		return nil, err
	}
	gatling.ObjectMeta.Name = serviceName
	gatling.Spec.PodSpec.GatlingImage = imgURL
	gatling.Spec.TestScenarioSpec = scenarioSpec.TestScenarioSpec
	return gatling, nil
}

/*
loadGatlingReportFromCloudStorage fetch gatling report and parse to gatling report object.

Fetch gatling report path in cloud storage from gatling object. And fetch gatling report bytes from cloud storage.
And parse gatling report bytes to gatling report object.
*/
func loadGatlingReportFromCloudStorage(
	ctx context.Context,
	op cloudStorageOperator,
	cl ctrlClient.Client,
	gatling *gatlingv1alpha1.Gatling,
) (*gatlingTools.GatlingReport, error) {
	reportStorageFolderPath, err := gatlingTools.GetGatlingReportStoragePath(ctx, cl, gatling)
	reportStorageObjectPath := reportStorageFolderPath + "/js/global_stats.json"
	if err != nil {
		return nil, fmt.Errorf("failed to get gatling report storage path, %w\n", err)
	}
	fetchedReportBytes, err := op.Fetch(ctx, reportStorageObjectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch report, %w\n", err)
	}
	gatlingReport, err := gatlingTools.BytesToGatlingReport(fetchedReportBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse gatling report, %w\n", err)
	}
	return gatlingReport, nil
}

/*
writeReportToSpreadsheets write loadtest report to spreadsheet.

Set column header name and add each row which has loadtest report value.
The sheet create by date by service. Set column header is called only once when sheet created.
*/
func writeReportToSpreadsheets(
	ctx context.Context,
	imageURL string,
	serviceConfig serviceConfig,
	scenarioSpec cfg.ScenarioSpec,
	gatlingReport *gatlingTools.GatlingReport,
	mRatio metricsUsageRatio,
) error {
	targetLatency := serviceConfig.targetLatency
	targetPercentile := serviceConfig.targetPercentile
	serviceName := serviceConfig.name

	op, err := sheetTools.NewSpreadsheetOperator(ctx, serviceConfig.spreadsheetId)
	if err != nil {
		return fmt.Errorf("failed to init spreadsheet operator, %w", err)
	}
	sheetTitle := fmt.Sprintf("%v-%v", scenarioSpec.Name, time.Now().Format("20060102"))
	targetSheet, err := op.FindSheet(sheetTitle)
	if err != nil && !errors.Is(err, &sheetTools.SheetNotFoundError{}) {
		return fmt.Errorf("unexpected error occured when FindSheet, %w", err)
	}
	if errors.Is(err, &sheetTools.SheetNotFoundError{}) {
		targetSheet, err = op.AddSheet(sheetTitle)
		if err != nil {
			return fmt.Errorf("failed to create new sheet, %w", err)
		}
		targetSheet, err = op.SetColumnHeader(targetSheet)
		if err != nil {
			return fmt.Errorf("failed to set cell name, %w", err)
		}
	}
	var targetLatencyFieldValue string
	if targetPercentile == 0 && targetLatency == 0 {
		targetLatencyFieldValue = "target latency not specified"
	} else {
		targetLatencyFieldValue = fmt.Sprintf("percentile %v, latency %vms", targetPercentile, targetLatency)
	}
	commonSettingValue := sheetTools.NewLoadtestCommonSettingRow(imageURL, serviceName, targetLatencyFieldValue)
	targetSheet, err = op.SetLoadtestCommonSettingValue(commonSettingValue, targetSheet)
	if err != nil {
		return fmt.Errorf("failed to set loadtest common setting value %w", err)
	}

	concurrency, duration, condition, err := gatlingTools.ExtractLoadtestConditionToReport(
		scenarioSpec.TestScenarioSpec,
	)
	if err != nil {
		return fmt.Errorf("failed to parse loadtest condition %w", err)
	}

	row := sheetTools.NewLoadtestReportRow(
		scenarioSpec.SubName,
		condition,
		duration,
		concurrency,
		gatlingReport.MaxResponseTime.Ok,
		gatlingReport.MeanResponseTime.Ok,
		gatlingReport.FiftiethPercentiles.Ok,
		gatlingReport.SeventyFifthPercentiles.Ok,
		gatlingReport.NintyFifthPercentiles.Ok,
		gatlingReport.NintyNinthPercentiles.Ok,
		gatlingReport.Failed.Percentage,
		gatlingReport.UnderEightHundredMilliSec.Percentage,
		gatlingReport.BetweenFromEightHundredToOneThousandTwoHundredMilliSec.Percentage,
		gatlingReport.OverOneThousandTwoHundredMilliSec.Percentage,
		mRatio.cpu*100,    // conv ratio to percentage
		mRatio.memory*100, // conv ratio to percentage
	)
	_, err = op.AppendLoadtestReportRow(row, targetSheet)
	if err != nil {
		return err
	}
	return nil
}

// notifyLoadtestResult call notifyOperator Notify method.
func notifyLoadtestResult(op notifyOperator, data string) error {
	err := op.Notify(data)
	if err != nil {
		return err
	}
	return nil
}

/*
checkContinueToExec returns checkContinueToExecResult which has shouldContinue boolean flag.

If shouldContinue is false, loadtest per service will be interrupted.

Some cases of shouldContinue is false are below.
- config.yaml parameter failFast is true and gatlingReport.Failed.Percentage is more than 0
- config.yaml parameter targetLatency and targetPercentile specified,
and gatlingReport target Percentile latency value is more than targetLatency.
*/
func checkContinueToExec(
	serviceConfig serviceConfig,
	gatlingReport gatlingTools.GatlingReport,
) (*checkContinueToExecResult, error) {
	failFast := serviceConfig.failFast
	targetLatency := serviceConfig.targetLatency
	targetPercentile := serviceConfig.targetPercentile

	// Check failed percentage
	if failFast {
		if gatlingReport.Failed.Percentage > 0 {
			return &checkContinueToExecResult{
				shouldContinue: false,
				message:        "failed percentage greater than 0",
			}, nil
		}
	}
	// nolint:lll // If targetLatency and targetPercentile field value exist, check whether latency in gatling report is larger than target latency or not.
	// these field are already validated when cli loaded config.yaml.
	if targetLatency != 0 && targetPercentile != 0 {
		resultLatency, err := gatlingReport.GetPercentileLatency(targetPercentile)
		if err != nil {
			return nil, fmt.Errorf("failed to get specified percentile latency %v", err)
		}
		if resultLatency > targetLatency {
			return &checkContinueToExecResult{
				shouldContinue: false,
				message: fmt.Sprintf(
					"latency below specified target, target: %v, result: %v",
					targetLatency,
					resultLatency,
				),
			}, nil
		}
	}
	return &checkContinueToExecResult{
		shouldContinue: true,
		message:        "",
	}, nil
}

// extractServiceConfig extract cfg.Service struct field use for logging service metadata and so on.
func extractServiceConfig(s cfg.Service) serviceConfig {
	return serviceConfig{
		name:             s.Name,
		spreadsheetId:    s.SpreadsheetId,
		failFast:         s.FailFast,
		targetLatency:    s.TargetLatency,
		targetPercentile: s.TargetPercentile,
	}
}
