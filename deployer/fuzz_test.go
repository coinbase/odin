package deployer

import (
	"testing"

	"github.com/coinbase/odin/deployer/models"
	"github.com/coinbase/step/utils/to"
	fuzz "github.com/google/gofuzz"
	"github.com/stretchr/testify/assert"
)

func Test_Release_Basic_Fuzz(t *testing.T) {
	for i := 0; i < 50; i++ {
		f := fuzz.New()
		var release models.Release
		f.Fuzz(&release) // myInt gets a random value.

		assertNoPanic(t, &release)
	}
}

func Test_Release_Basic_Service_Fuzz(t *testing.T) {
	for i := 0; i < 50; i++ {
		f := fuzz.New()
		release := models.MockRelease(t)
		f.Fuzz(release.Services["web"]) // myInt gets a random value.

		assertNoPanic(t, release)
	}
}

func Test_Release_Basic_Autoscaling_Fuzz(t *testing.T) {
	for i := 0; i < 50; i++ {
		f := fuzz.New()
		release := models.MockRelease(t)
		f.Fuzz(release.Services["web"].Autoscaling) // myInt gets a random value.

		assertNoPanic(t, release)
	}
}

func Test_Release_Basic_Policies_Fuzz(t *testing.T) {
	for i := 0; i < 25; i++ {
		f := fuzz.New()
		release := models.MockRelease(t)
		f.Fuzz(release.Services["web"].Autoscaling.Policies[0]) // myInt gets a random value.
		release.Services["web"].Autoscaling.Policies[0].Type = to.Strp("cpu_scale_up")
		assertNoPanic(t, release)
	}

	for i := 0; i < 25; i++ {
		f := fuzz.New()
		release := models.MockRelease(t)
		f.Fuzz(release.Services["web"].Autoscaling.Policies[0]) // myInt gets a random value.
		release.Services["web"].Autoscaling.Policies[0].Type = to.Strp("cpu_scale_down")
		assertNoPanic(t, release)
	}

}

func Test_Release_Basic_LifeCycle_Fuzz(t *testing.T) {
	for i := 0; i < 50; i++ {
		f := fuzz.New()
		release := models.MockRelease(t)
		f.Fuzz(release.LifeCycleHooks["TermHook"]) // myInt gets a random value.

		assertNoPanic(t, release)
	}
}

func assertNoPanic(t *testing.T, release *models.Release) {
	release.AwsAccountID = to.Strp("0000000")
	stateMachine := createTestStateMachine(t, models.MockAwsClients(release))

	_, err := stateMachine.ExecuteToMap(release)
	if err != nil {
		assert.NotRegexp(t, "Panic", err.Error())
	}

	le := stateMachine.LastOutput()

	assert.NotRegexp(t, "Panic", le)
}
