package runner

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/kubeshop/testkube-executor-tracetest/pkg/model"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/envs"
	"github.com/kubeshop/testkube/pkg/executor/content"
	outputPkg "github.com/kubeshop/testkube/pkg/executor/output"
	"github.com/kubeshop/testkube/pkg/executor/runner"
	"github.com/kubeshop/testkube/pkg/executor/secret"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewRunner() (*TracetestRunner, error) {
	outputPkg.PrintLog(fmt.Sprintf("%s [TracetestRunner]: Preparing Runner", ui.IconTruck))

	params, err := envs.LoadTestkubeVariables()
	if err != nil {
		return nil, fmt.Errorf("could not initialize Testkube variables: %w", err)
	}

	return &TracetestRunner{
		Fetcher:       content.NewFetcher(""),
		Params:        params,
		coreExecutor:  &tracetestCoreExecutor{},
		cloudExecutor: &tracetestCloudExecutor{},
	}, nil
}

type TracetestRunner struct {
	Fetcher       content.ContentFetcher
	Params        envs.Params
	coreExecutor  TracetestCLIExecutor
	cloudExecutor TracetestCLIExecutor
}

type TracetestCLIExecutor interface {
	RequiredEnvVars() []string
	HasEnvVarsDefined(*secret.EnvManager) bool
	Execute(*secret.EnvManager, testkube.Execution, string) (model.Result, error)
}

func (r *TracetestRunner) Run(execution testkube.Execution) (testkube.ExecutionResult, error) {
	outputPkg.PrintLog(fmt.Sprintf("%s [TracetestRunner]: Preparing test run", ui.IconTruck))

	envManager := secret.NewEnvManagerWithVars(execution.Variables)
	envManager.GetVars(envManager.Variables)

	// Get execution content file path
	testFilePath, err := getContentPath(r.Params.DataDir, execution.Content, r.Fetcher)
	if err != nil {
		outputPkg.PrintLog(fmt.Sprintf("%s [TracetestRunner]: Error fetching the content file", ui.IconCross))
		return testkube.ExecutionResult{}, err
	}

	// Execute a test
	cliExecutor, err := r.getCLIExecutor(envManager)
	if err != nil {
		return testkube.ExecutionResult{}, nil
	}

	// Execute test and format output
	result, err := cliExecutor.Execute(envManager, execution, testFilePath)
	if err != nil {
		return result.ToFailedExecutionResult(err), nil
	}

	return result.ToSuccessfulExecutionResult(), nil
}

// GetType returns runner type
func (r *TracetestRunner) GetType() runner.Type {
	return runner.TypeMain
}

func (r *TracetestRunner) getCLIExecutor(envManager *secret.EnvManager) (TracetestCLIExecutor, error) {
	if r.cloudExecutor.HasEnvVarsDefined(envManager) {
		return r.cloudExecutor, nil
	}

	if r.coreExecutor.HasEnvVarsDefined(envManager) {
		return r.coreExecutor, nil
	}

	outputPkg.PrintLog(fmt.Sprintf("%s [TracetestRunner]: Could not find variables to run the test with Tracetest or Tracetest Cloud.", ui.IconCross))
	outputPkg.PrintLog(fmt.Sprintf("[TracetestRunner]: Please define the [%s] variables to run a test with Tracetest", strings.Join(r.cloudExecutor.RequiredEnvVars(), ", ")))
	outputPkg.PrintLog(fmt.Sprintf("[TracetestRunner]: Or define the [%s] variables to run a test with Tracetest Core", strings.Join(r.coreExecutor.RequiredEnvVars(), ", ")))
	return nil, fmt.Errorf("could not find variables to run the test with Tracetest or Tracetest Cloud")
}

// Get variable from EnvManager
func getVariable(envManager *secret.EnvManager, variableName string) (string, error) {
	return getVariableWithWarning(envManager, variableName, true)
}

func getOptionalVariable(envManager *secret.EnvManager, variableName string) (string, error) {
	return getVariableWithWarning(envManager, variableName, false)
}

func getVariableWithWarning(envManager *secret.EnvManager, variableName string, required bool) (string, error) {
	v, ok := envManager.Variables[variableName]

	warningMessage := fmt.Sprintf("%s [TracetestRunner]: %s variable was not found", ui.IconCross, variableName)
	if !required {
		warningMessage = fmt.Sprintf("[TracetestRunner]: %s variable was not found, assuming empty value", variableName)
	}

	if !ok {
		outputPkg.PrintLog(warningMessage)
		return "", fmt.Errorf(variableName + " variable was not found")
	}

	return strings.ReplaceAll(v.Value, "\"", ""), nil
}

// Get execution content file path
func getContentPath(dataDir string, content *testkube.TestContent, fetcher content.ContentFetcher) (string, error) {
	// Check that the data dir exists
	_, err := os.Stat(dataDir)
	if errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("Data directory '%s' does not exist", dataDir)
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
