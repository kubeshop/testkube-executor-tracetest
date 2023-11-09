package runner

import (
	"github.com/kubeshop/testkube-executor-tracetest/pkg/model"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/executor"
	"github.com/kubeshop/testkube/pkg/executor/secret"
)

const (
	TRACETEST_ENDPOINT_VAR        = "TRACETEST_ENDPOINT"
	TRACETEST_OUTPUT_ENDPOINT_VAR = "TRACETEST_OUTPUT_ENDPOINT"
)

type tracetestCoreExecutor struct{}

var _ TracetestCLIExecutor = (*tracetestCoreExecutor)(nil)

func (e *tracetestCoreExecutor) RequiredEnvVars() []string {
	return []string{TRACETEST_ENDPOINT_VAR}
}

func (e *tracetestCoreExecutor) HasEnvVarsDefined(envManager *secret.EnvManager) bool {
	_, hasEndpointVar := envManager.Variables[TRACETEST_ENDPOINT_VAR]
	return hasEndpointVar
}

func (e *tracetestCoreExecutor) Execute(envManager *secret.EnvManager, execution testkube.Execution, testFilePath string) (model.Result, error) {
	// Get TRACETEST_ENDPOINT from execution variables
	tracetestEndpoint, err := getVariable(envManager, TRACETEST_ENDPOINT_VAR)
	if err != nil {
		return model.Result{}, err
	}

	// Get TRACETEST_OUTPUT_ENDPOINT from execution variables
	tracetestOutputEndpoint, _ := getOptionalVariable(envManager, TRACETEST_OUTPUT_ENDPOINT_VAR)

	// Prepare args for test run command
	args := []string{
		"run", "test", "--server-url", tracetestEndpoint, "--file", testFilePath, "--output", "pretty",
	}
	// Pass additional execution arguments to tracetest
	// args = append(args, execution.Args...)

	// Run tracetest test from definition file
	output, err := executor.Run("", "tracetest", envManager, args...)

	result := model.Result{
		Output:         string(output),
		ServerEndpoint: tracetestEndpoint,
		OutputEndpoint: tracetestOutputEndpoint,
	}

	return result, err
}
