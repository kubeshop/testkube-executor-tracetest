package runner

import (
	"fmt"

	"github.com/kubeshop/testkube-executor-tracetest/pkg/model"

	"github.com/kubeshop/testkube-executor-tracetest/pkg/command"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/executor"
	outputPkg "github.com/kubeshop/testkube/pkg/executor/output"
	"github.com/kubeshop/testkube/pkg/executor/secret"
	"github.com/kubeshop/testkube/pkg/ui"
)

const (
	TRACETEST_TOKEN_VAR        = "TRACETEST_TOKEN"
	TRACETEST_ORGANIZATION_VAR = "TRACETEST_ORGANIZATION"
	TRACETEST_ENVIRONMENT_VAR  = "TRACETEST_ENVIRONMENT"
)

type tracetestCloudExecutor struct{}

var _ TracetestCLIExecutor = (*tracetestCloudExecutor)(nil)

func (e *tracetestCloudExecutor) RequiredEnvVars() []string {
	return []string{TRACETEST_TOKEN_VAR, TRACETEST_ORGANIZATION_VAR, TRACETEST_ENVIRONMENT_VAR}
}

func (e *tracetestCloudExecutor) HasEnvVarsDefined(envManager *secret.EnvManager) bool {
	_, hasTokenVar := envManager.Variables[TRACETEST_TOKEN_VAR]
	_, hasOrganizationVar := envManager.Variables[TRACETEST_ORGANIZATION_VAR]
	_, hasEnvironmentVar := envManager.Variables[TRACETEST_ENVIRONMENT_VAR]

	return hasTokenVar && hasOrganizationVar && hasEnvironmentVar
}

func (e *tracetestCloudExecutor) Execute(envManager *secret.EnvManager, execution testkube.Execution, testFilePath string) (model.Result, error) {
	tracetestToken, err := getVariable(envManager, TRACETEST_TOKEN_VAR)
	if err != nil {
		return model.Result{}, err
	}

	tracetestOrganization, err := getVariable(envManager, TRACETEST_ORGANIZATION_VAR)
	if err != nil {
		return model.Result{}, err
	}

	tracetestEnvironment, err := getVariable(envManager, TRACETEST_ENVIRONMENT_VAR)
	if err != nil {
		return model.Result{}, err
	}

	// setup config with API key
	outputPkg.PrintLog(fmt.Sprintf("%s [TracetestRunner]: Configuring Tracetest CLI with Token", ui.IconTruck))
	_, err = command.Run("tracetest", "configure", "--token", tracetestToken, "--organization", tracetestOrganization, "--environment", tracetestEnvironment)
	if err != nil {
		outputPkg.PrintLog(fmt.Sprintf("%s [TracetestRunner]: Failed to configure Tracetest CLI", ui.IconCross))
		return model.Result{}, err
	}

	// Prepare args for test run command
	args := []string{
		"run", "test", "--file", testFilePath, "--output", "pretty",
	}
	// Pass additional execution arguments to tracetest
	// args = append(args, execution.Args...)

	// Run tracetest test from definition file
	output, err := executor.Run("", "tracetest", envManager, args...)

	result := model.Result{
		Output:         string(output),
		ServerEndpoint: "https://app.tracetest.io",
		OutputEndpoint: "",
	}

	return result, err
}
