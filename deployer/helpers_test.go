package deployer

import (
	"testing"

	"github.com/coinbase/odin/aws"
	"github.com/coinbase/odin/deployer/models"
	"github.com/coinbase/step/machine"
	"github.com/stretchr/testify/assert"
)

func assertSuccessfulExecution(t *testing.T, release *models.Release) {
	stateMachine := createTestStateMachine(t, models.MockAwsClients(release))

	output, err := stateMachine.ExecuteToMap(release)

	assert.NoError(t, err)
	assert.Equal(t, true, output["success"])
	assert.NotRegexp(t, "error", stateMachine.LastOutput())

	assert.Equal(t, stateMachine.ExecutionPath(), []string{
		"ValidateFn",
		"Validate",
		"LockFn",
		"Lock",
		"ValidateResourcesFn",
		"ValidateResources",
		"DeployFn",
		"Deploy",
		"WaitForDeploy",
		"WaitForHealthy",
		"CheckHealthyFn",
		"CheckHealthy",
		"Healthy?",
		"CleanUpSuccessFn",
		"CleanUpSuccess",
		"Success",
	})
}

//////////
// CREATING THE STATE MACHINE
//////////

func createTestStateMachine(t *testing.T, awsc aws.Clients) *machine.StateMachine {
	tm := CreateTaskFunctinons(awsc)

	stateMachine, err := StateMachineWithTaskHandlers(tm)
	assert.NoError(t, err)

	return stateMachine
}
