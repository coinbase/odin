package deployer

import (
	"fmt"
	"testing"

	"github.com/coinbase/odin/aws/mocks"
	"github.com/coinbase/odin/deployer/models"
	"github.com/coinbase/step/utils/to"
	"github.com/stretchr/testify/assert"
)

///////////////
// Successful Tests
///////////////

func Test_Successful_Execution_Works(t *testing.T) {
	// Should end in Alert Bad Thing Happened State
	release := models.MockRelease(t)
	assertSuccessfulExecution(t, release)
}

func Test_Successful_Execution_Works_With_Minimal_Release(t *testing.T) {
	// Should end in Alert Bad Thing Happened State
	release := models.MockMinimalRelease(t)
	assertSuccessfulExecution(t, release)
}

func Test_Successful_Execution_Works_With_UserDataTemplate(t *testing.T) {
	// Should end in Alert Bad Thing Happened State
	release := models.MockMinimalRelease(t)
	release.UserData = to.Strp("{{RELEASE_ID}}\n{{PROJECT_NAME}}\n{{CONFIG_NAME}}\n{{SERVICE_NAME}}\n")
	s1 := release.Services["web"]
	s1.SetDefaults(release, "web")
	assert.Equal(t, *s1.UserData(), fmt.Sprintf("%v\n%v\n%v\nweb\n", *release.ReleaseID, *release.ProjectName, *release.ConfigName))
	assertSuccessfulExecution(t, release)
}

///////////////
// Unsuccessful Tests
///////////////

func Test_UnsuccessfulDeploy_Execution_Works(t *testing.T) {
	release := models.MockRelease(t)
	release.Timeout = to.Intp(-10) // This will cause immediate timeout

	// Should end in Alert Bad Thing Happened State
	stateMachine := createTestStateMachine(t, models.MockAwsClients(release))

	output, err := stateMachine.ExecuteToMap(release)

	assert.Error(t, err)
	assert.Equal(t, "FailureClean", output["Error"])

	assert.Equal(t, []string{
		"ValidateFn",
		"Validate",
		"LockFn",
		"Lock",
		"ValidateResourcesFn",
		"ValidateResources",
		"DeployFn",
		"Deploy",
		"ReleaseLockFailureFn",
		"ReleaseLockFailure",
		"FailureClean",
	}, stateMachine.ExecutionPath())
}

///////////////
// MACHINE FetchDeploy INTERGATION TESTS
///////////////

func Test_Execution_FetchDeploy_BadInputError(t *testing.T) {
	// Should end in clean state as nothing has happened yet
	stateMachine := createTestStateMachine(t, models.MockAwsClients(models.MockRelease(t)))

	output, err := stateMachine.ExecuteToMap(struct{}{})

	assert.Error(t, err)
	assert.Equal(t, "FailureClean", output["Error"])

	assert.Equal(t, stateMachine.ExecutionPath(), []string{
		"ValidateFn",
		"Validate",
		"FailureClean",
	})
}

func Test_Execution_FetchDeploy_UnkownKeyInput(t *testing.T) {
	// Should end in clean state as nothing has happened yet
	stateMachine := createTestStateMachine(t, models.MockAwsClients(models.MockRelease(t)))

	output, err := stateMachine.ExecuteToMap(struct{ Unkown string }{Unkown: "asd"})

	assert.Error(t, err)
	assert.Equal(t, "FailureClean", output["Error"])
	assert.Regexp(t, "unknown field", stateMachine.LastOutput())

	assert.Equal(t, stateMachine.ExecutionPath(), []string{
		"ValidateFn",
		"Validate",
		"FailureClean",
	})
}

func Test_Execution_FetchDeploy_BadInputError_Unamarshalling(t *testing.T) {
	// Should end in clean state as nothing has happened yet
	stateMachine := createTestStateMachine(t, models.MockAwsClients(models.MockRelease(t)))

	output, err := stateMachine.ExecuteToMap(struct{ Subnets string }{Subnets: ""})

	assert.Error(t, err)
	assert.Equal(t, "FailureClean", output["Error"])

	assert.Equal(t, stateMachine.ExecutionPath(), []string{
		"ValidateFn",
		"Validate",
		"FailureClean",
	})
}

func Test_Execution_FetchDeploy_LockError(t *testing.T) {
	release := models.MockRelease(t)

	// Should retry a few times, then end in clean state as nothing was created
	awsClients := models.MockAwsClients(release)

	// Force a lock error by making it look like it was already aquired
	awsClients.S3.AddGetObject(*release.LockPath(), `{"uuid": "already"}`, nil)

	stateMachine := createTestStateMachine(t, awsClients)

	output, err := stateMachine.ExecuteToMap(release)

	assert.Error(t, err)
	assert.Equal(t, "FailureClean", output["Error"])

	assert.Equal(t, stateMachine.ExecutionPath(), []string{
		"ValidateFn",
		"Validate",
		"LockFn",
		"Lock",
		"FailureClean",
	})
}

func Test_Execution_CheckHealthy_HaltError_WithTermination(t *testing.T) {
	// Should end in Alert Bad Thing Happened State
	release := models.MockRelease(t)
	maws := models.MockAwsClients(release)
	maws.ASG.DescribeAutoScalingGroupsPageResp = nil

	termingASG := mocks.MakeMockASG("odin", *release.ProjectName, *release.ConfigName, "web", "Old release")
	termingASG.Instances[0].LifecycleState = to.Strp("Terminating")

	maws.ASG.AddASG(termingASG)

	stateMachine := createTestStateMachine(t, maws)

	_, err := stateMachine.ExecuteToMap(release)

	assert.Error(t, err)
	assert.Regexp(t, "HaltError", stateMachine.LastOutput())
	assert.Regexp(t, "success\":false", stateMachine.LastOutput())

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
		"CleanUpFailureFn",
		"CleanUpFailure",
		"ReleaseLockFailureFn",
		"ReleaseLockFailure",
		"FailureClean",
	})
}

func Test_Execution_CheckHealthy_Never_Healthy_ELB(t *testing.T) {
	// Should end in Alert Bad Thing Happened State
	release := models.MockRelease(t)

	maws := models.MockAwsClients(release)
	maws.ELB.DescribeInstanceHealthResp["web-elb"] = &mocks.DescribeInstanceHealthResponse{}

	stateMachine := createTestStateMachine(t, maws)

	_, err := stateMachine.ExecuteToMap(release)

	assert.Error(t, err)

	ep := stateMachine.ExecutionPath()
	assert.Equal(t, []string{
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
	}, ep[0:12])

	assert.Equal(t, []string{
		"CleanUpFailureFn",
		"CleanUpFailure",
		"ReleaseLockFailureFn",
		"ReleaseLockFailure",
		"FailureClean",
	}, ep[len(ep)-5:len(ep)])

	assert.Regexp(t, "Timeout", stateMachine.LastOutput())
	assert.Regexp(t, "success\":false", stateMachine.LastOutput())
}

func Test_Execution_CheckHealthy_Never_Healthy_TG(t *testing.T) {
	// Should end in Alert Bad Thing Happened State
	release := models.MockRelease(t)

	maws := models.MockAwsClients(release)
	maws.ALB.DescribeTargetHealthResp["web-elb-target"] = &mocks.DescribeTargetHealthResponse{}

	stateMachine := createTestStateMachine(t, maws)

	_, err := stateMachine.ExecuteToMap(release)

	assert.Error(t, err)

	ep := stateMachine.ExecutionPath()
	assert.Equal(t, []string{
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
	}, ep[0:12])

	assert.Equal(t, []string{
		"CleanUpFailureFn",
		"CleanUpFailure",
		"ReleaseLockFailureFn",
		"ReleaseLockFailure",
		"FailureClean",
	}, ep[len(ep)-5:len(ep)])

	assert.Regexp(t, "Timeout", stateMachine.LastOutput())
	assert.Regexp(t, "success\":false", stateMachine.LastOutput())
}
