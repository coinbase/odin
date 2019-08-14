package deployer

import (
	"testing"

	"github.com/aws/aws-sdk-go/service/autoscaling"

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

///////////////
// Unsuccessful Tests
///////////////

func Test_UnsuccessfulDeploy_Bad_Userdata_SHA(t *testing.T) {
	// Should end in Alert Bad Thing Happened State
	release := models.MockMinimalRelease(t)

	// Should end in Alert Bad Thing Happened State
	stateMachine := createTestStateMachine(t, models.MockAwsClients(release))
	release.UserDataSHA256 = to.Strp("asfhjoias")

	exec, err := stateMachine.Execute(release)
	output := exec.Output

	assert.Error(t, err)
	assert.Equal(t, "FailureClean", output["Error"])

	assert.Equal(t, []string{
		"Validate",
		"FailureClean",
	}, exec.Path())
}

func Test_UnsuccessfulDeploy_Execution_Works(t *testing.T) {
	release := models.MockRelease(t)
	release.Timeout = to.Intp(-10) // This will cause immediate timeout

	// Should end in Alert Bad Thing Happened State
	stateMachine := createTestStateMachine(t, models.MockAwsClients(release))

	exec, err := stateMachine.Execute(release)
	output := exec.Output

	assert.Error(t, err)
	assert.Equal(t, "FailureClean", output["Error"])

	assert.Equal(t, []string{
		"Validate",
		"Lock",
		"ValidateResources",
		"Deploy",
		"ReleaseLockFailure",
		"FailureClean",
	}, exec.Path())
}

///////////////
// MACHINE FetchDeploy INTERGATION TESTS
///////////////

func Test_Execution_FetchDeploy_BadInputError(t *testing.T) {
	// Should end in clean state as nothing has happened yet
	stateMachine := createTestStateMachine(t, models.MockAwsClients(models.MockRelease(t)))

	exec, err := stateMachine.Execute(struct{}{})
	output := exec.Output

	assert.Error(t, err)
	assert.Equal(t, "FailureClean", output["Error"])

	assert.Equal(t, exec.Path(), []string{
		"Validate",
		"FailureClean",
	})
}

func Test_Execution_FetchDeploy_UnkownKeyInput(t *testing.T) {
	// Should end in clean state as nothing has happened yet
	stateMachine := createTestStateMachine(t, models.MockAwsClients(models.MockRelease(t)))

	exec, err := stateMachine.Execute(struct{ Unkown string }{Unkown: "asd"})
	output := exec.Output

	assert.Error(t, err)
	assert.Equal(t, "FailureClean", output["Error"])
	assert.Regexp(t, "unknown field", exec.LastOutputJSON)

	assert.Equal(t, exec.Path(), []string{
		"Validate",
		"FailureClean",
	})
}

func Test_Execution_FetchDeploy_BadInputError_Unamarshalling(t *testing.T) {
	// Should end in clean state as nothing has happened yet
	stateMachine := createTestStateMachine(t, models.MockAwsClients(models.MockRelease(t)))

	exec, err := stateMachine.Execute(struct{ Subnets string }{Subnets: ""})
	output := exec.Output

	assert.Error(t, err)
	assert.Equal(t, "FailureClean", output["Error"])

	assert.Equal(t, exec.Path(), []string{
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

	exec, err := stateMachine.Execute(release)
	output := exec.Output

	assert.Error(t, err)
	assert.Equal(t, "FailureClean", output["Error"])

	assert.Equal(t, exec.Path(), []string{
		"Validate",
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

	exec, err := stateMachine.Execute(release)

	assert.Error(t, err)
	assert.Regexp(t, "HaltError", exec.LastOutputJSON)
	assert.Regexp(t, "success\": false", exec.LastOutputJSON)

	assert.Equal(t, exec.Path(), []string{
		"Validate",
		"Lock",
		"ValidateResources",
		"Deploy",
		"WaitForDeploy",
		"WaitForHealthy",
		"CheckHealthy",
		"CleanUpFailure",
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

	exec, err := stateMachine.Execute(release)

	assert.Error(t, err)

	ep := exec.Path()
	assert.Equal(t, []string{
		"Validate",
		"Lock",
		"ValidateResources",
		"Deploy",
		"WaitForDeploy",
		"WaitForHealthy",
		"CheckHealthy"}, ep[0:7])

	assert.Equal(t, []string{
		"CleanUpFailure",
		"ReleaseLockFailure",
		"FailureClean",
	}, ep[len(ep)-3:len(ep)])

	assert.Regexp(t, "Timeout", exec.LastOutputJSON)
	assert.Regexp(t, "success\": false", exec.LastOutputJSON)
}

func Test_Execution_CheckHealthy_Never_Healthy_TG(t *testing.T) {
	// Should end in Alert Bad Thing Happened State
	release := models.MockRelease(t)

	maws := models.MockAwsClients(release)
	maws.ALB.DescribeTargetHealthResp["web-elb-target"] = &mocks.DescribeTargetHealthResponse{}

	stateMachine := createTestStateMachine(t, maws)

	exec, err := stateMachine.Execute(release)

	assert.Error(t, err)

	ep := exec.Path()
	assert.Equal(t, []string{
		"Validate",
		"Lock",
		"ValidateResources",
		"Deploy",
		"WaitForDeploy",
		"WaitForHealthy",
		"CheckHealthy"}, ep[0:7])

	assert.Equal(t, []string{
		"CleanUpFailure",
		"ReleaseLockFailure",
		"FailureClean",
	}, ep[len(ep)-3:len(ep)])

	assert.Regexp(t, "Timeout", exec.LastOutputJSON)
	assert.Regexp(t, "success\": false", exec.LastOutputJSON)
}

func Test_Execution_CleanupSuccess_DetachError(t *testing.T) {
	// Should try 10 times to detach
	release := models.MockRelease(t)

	maws := models.MockAwsClients(release)
	maws.ASG.DescribeLoadBalancerTargetGroupsOutput = &autoscaling.DescribeLoadBalancerTargetGroupsOutput{
		LoadBalancerTargetGroups: []*autoscaling.LoadBalancerTargetGroupState{
			&autoscaling.LoadBalancerTargetGroupState{
				LoadBalancerTargetGroupARN: to.Strp("arn"),
				State: to.Strp("aaa"),
			},
		},
	}

	stateMachine := createTestStateMachine(t, maws)

	exec, err := stateMachine.Execute(release)

	assert.Error(t, err)

	ep := exec.Path()
	assert.Equal(t, []string{
		"Validate",
		"Lock",
		"ValidateResources",
		"Deploy",
		"WaitForDeploy",
		"WaitForHealthy",
		"CheckHealthy",
		"Healthy?",
		"CleanUpSuccess",
		"CleanUpSuccess",
		"CleanUpSuccess",
		"CleanUpSuccess",
		"CleanUpSuccess",
		"CleanUpSuccess",
		"CleanUpSuccess",
		"CleanUpSuccess",
		"CleanUpSuccess",
		"CleanUpSuccess",
		"CleanUpSuccess",
		"FailureDirty",
	}, ep)

	assert.Regexp(t, "DetachError", exec.LastOutputJSON)
}
