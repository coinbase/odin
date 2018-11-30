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

	exec, err := stateMachine.Execute(release)
	output := exec.Output

	assert.NoError(t, err)
	assert.Equal(t, true, output["success"])
	assert.NotRegexp(t, "error", exec.LastOutputJSON)

	assert.Equal(t, exec.Path(), []string{
		"Validate",
		"Lock",
		"ValidateResources",
		"Deploy",
		"WaitForDeploy",
		"WaitForHealthy",
		"CheckHealthy",
		"Healthy?",
		"CleanUpSuccess",
		"Success",
	})
}

//////////
// CREATING THE STATE MACHINE
//////////

func createTestStateMachine(t *testing.T, awsc aws.Clients) *machine.StateMachine {
	stateMachine, err := StateMachine()
	assert.NoError(t, err)

	err = stateMachine.SetTaskFnHandlers(CreateTaskFunctinons(awsc))
	assert.NoError(t, err)

	return stateMachine
}
