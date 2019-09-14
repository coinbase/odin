package deployer

import (
	"testing"

	"github.com/coinbase/odin/aws"
	"github.com/coinbase/odin/deployer/models"
	"github.com/coinbase/step/machine"
	"github.com/coinbase/step/utils/to"
	"github.com/stretchr/testify/assert"
)

func assertSuccessfulExecution(t *testing.T, release *models.Release) {
	awsc := models.MockAwsClients(release)
	stateMachine := createTestStateMachine(t, awsc)

	previousRelease := models.MockRelease(t)
	previousRelease.ReleaseID = to.Strp("old-release")
	models.AddReleaseS3Objects(awsc, previousRelease)

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
		"DetachForSuccess",
		"WaitDetachForSuccess",
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
