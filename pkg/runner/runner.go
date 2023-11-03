package runner

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/kubeshop/testkube-executor-tracetest/pkg/command"
	"github.com/kubeshop/testkube-executor-tracetest/pkg/model"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/envs"
	"github.com/kubeshop/testkube/pkg/executor"
	"github.com/kubeshop/testkube/pkg/executor/content"
	outputPkg "github.com/kubeshop/testkube/pkg/executor/output"
	"github.com/kubeshop/testkube/pkg/executor/runner"
	"github.com/kubeshop/testkube/pkg/executor/secret"
	"github.com/kubeshop/testkube/pkg/ui"
)

const TRACETEST_ENDPOINT_VAR = "TRACETEST_ENDPOINT"
const TRACETEST_OUTPUT_ENDPOINT_VAR = "TRACETEST_OUTPUT_ENDPOINT"
const TRACETEST_API_KEY_VAR = "TRACETEST_API_KEY"

func NewRunner() (*TracetestRunner, error) {
	outputPkg.PrintLog(fmt.Sprintf("%s [TracetestRunner]: Preparing Runner", ui.IconTruck))

	params, err := envs.LoadTestkubeVariables()
	if err != nil {
		return nil, fmt.Errorf("could not initialize Testkube variables: %w", err)
	}

	return &TracetestRunner{
		Fetcher: content.NewFetcher(""),
		Params:  params,
	}, nil
}

type TracetestRunner struct {
	Fetcher content.ContentFetcher
	Params  envs.Params
}

func (r *TracetestRunner) Run(execution testkube.Execution) (result testkube.ExecutionResult, err error) {
	outputPkg.PrintLog(fmt.Sprintf("%s [TracetestRunner]: Preparing test run", ui.IconTruck))

	envManager := secret.NewEnvManagerWithVars(execution.Variables)
	envManager.GetVars(envManager.Variables)

	// Get TRACETEST_ENDPOINT from execution variables
	tracetestEndpoint, err := getVariable(envManager, TRACETEST_ENDPOINT_VAR)
	if err != nil {
		outputPkg.PrintLog(fmt.Sprintf("%s [TracetestRunner]: TRACETEST_ENDPOINT variable was not found", ui.IconCross))
		return result, err
	}

	// Get TRACETEST_OUTPUT_ENDPOINT from execution variables
	tracetestOutputEndpoint, err := getVariable(envManager, TRACETEST_OUTPUT_ENDPOINT_VAR)
	if err != nil {
		outputPkg.PrintLog("[TracetestRunner]: TRACETEST_OUTPUT_ENDPOINT variable was not found, assuming empty value")
	}

	tracetestApiKey, err := getVariable(envManager, TRACETEST_API_KEY_VAR)
	if err != nil {
		outputPkg.PrintLog("[TracetestRunner]: TRACETEST_API_KEY variable was not found, assuming empty value")
	}

	// Get execution content file path
	testFilePath, err := getContentPath(r.Params.DataDir, execution.Content, r.Fetcher)
	if err != nil {
		outputPkg.PrintLog(fmt.Sprintf("%s [TracetestRunner]: Error fetching the content file", ui.IconCross))
		return result, err
	}

	var output []byte
	if tracetestApiKey == "" {
		output, err = executeTestWithTracetestCore(envManager, execution, testFilePath, tracetestEndpoint)
	} else {
		output, err = executeTestWithTracetestCloud(envManager, execution, testFilePath, tracetestApiKey)
	}

	runResult := model.Result{Output: string(output), ServerEndpoint: tracetestEndpoint, OutputEndpoint: tracetestOutputEndpoint}

	if err != nil {
		result.ErrorMessage = runResult.GetOutput()
		result.Output = runResult.GetOutput()
		result.Status = testkube.ExecutionStatusFailed
		return result, nil
	}

	result.Output = runResult.GetOutput()
	result.Status = runResult.GetStatus()
	return result, nil
}

// GetType returns runner type
func (r *TracetestRunner) GetType() runner.Type {
	return runner.TypeMain
}

func executeTestWithTracetestCloud(envManager *secret.EnvManager, execution testkube.Execution, testFilePath, tracetestApiKey string) ([]byte, error) {
	// setup config with API key
	outputPkg.PrintLog(fmt.Sprintf("%s [TracetestRunner]: Configuring Tracetest CLI with API Key", ui.IconTruck))
	_, err := command.Run("tracetest", "configure", "--api-key", tracetestApiKey)
	if err != nil {
		return nil, err
	}

	// Prepare args for test run command
	args := []string{
		"run", "test", "--file", testFilePath, "--output", "pretty",
	}
	// Pass additional execution arguments to tracetest
	args = append(args, execution.Args...)

	// Run tracetest test from definition file
	return executor.Run("", "tracetest", envManager, args...)
}

func executeTestWithTracetestCore(envManager *secret.EnvManager, execution testkube.Execution, testFilePath, tracetestEndpoint string) ([]byte, error) {
	// Prepare args for test run command
	args := []string{
		"run", "test", "--server-url", tracetestEndpoint, "--file", testFilePath, "--output", "pretty",
	}
	// Pass additional execution arguments to tracetest
	args = append(args, execution.Args...)

	// Run tracetest test from definition file
	return executor.Run("", "tracetest", envManager, args...)
}

// Get variable from EnvManager
func getVariable(envManager *secret.EnvManager, variableName string) (string, error) {
	v, ok := envManager.Variables[variableName]
	if !ok {
		return "", fmt.Errorf(variableName + " variable was not found")
	}

	return strings.ReplaceAll(v.Value, "\"", ""), nil
}

// Get execution content file path
func getContentPath(dataDir string, content *testkube.TestContent, fetcher content.ContentFetcher) (string, error) {
	// Check that the data dir exists
	_, err := os.Stat(dataDir)
	if errors.Is(err, os.ErrNotExist) {
		return "", err
	}

	// Fetch execution content to file
	path, err := fetcher.Fetch(content)
	if err != nil {
		return "", err
	}

	if !content.IsFile() {
		return "", testkube.ErrTestContentTypeNotFile
	}

	return path, nil
}
