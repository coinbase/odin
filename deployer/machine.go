package deployer

// StateMachine returns the StateMachine
import (
	"github.com/coinbase/odin/aws"
	"github.com/coinbase/step/handler"
	"github.com/coinbase/step/machine"
)

// StateMachine returns
func StateMachine() (*machine.StateMachine, error) {
	stateMachine, err := machine.FromJSON([]byte(`{
    "Comment": "ASG Deployer",
    "StartAt": "ValidateFn",
    "States": {
      "ValidateFn": {
        "Type": "Pass",
        "Result": "Validate",
        "ResultPath": "$.Task",
        "Next": "Validate"
      },
      "Validate": {
        "Type": "Task",
        "Comment": "Validate and Set Defaults",
        "Next": "LockFn",
        "Catch": [
          {
            "Comment": "Bad Input, straight to Failure Clean, dont pass go dont collect $200",
            "ErrorEquals": ["BadReleaseError", "PanicError", "UnmarshalError"],
            "ResultPath": "$.error",
            "Next": "FailureClean"
          }
        ]
      },
      "LockFn": {
        "Type": "Pass",
        "Result": "Lock",
        "ResultPath": "$.Task",
        "Next": "Lock"
      },
      "Lock": {
        "Type": "Task",
        "Comment": "Grab Lock",
        "Next": "ValidateResourcesFn",
        "Catch": [
          {
            "Comment": "Bad Input, straight to Failure Clean, dont pass go dont collect $200",
            "ErrorEquals": ["LockExistsError"],
            "ResultPath": "$.error",
            "Next": "FailureClean"
          },
          {
            "Comment": "Release Lock if you created it",
            "ErrorEquals": ["LockError"],
            "ResultPath": "$.error",
            "Next": "ReleaseLockFailureFn"
          },
          {
            "Comment": "Panic is not good",
            "ErrorEquals": ["PanicError"],
            "ResultPath": "$.error",
            "Next": "FailureDirty"
          }
        ]
      },
      "ValidateResourcesFn": {
        "Type": "Pass",
        "Result": "ValidateResources",
        "ResultPath": "$.Task",
        "Next": "ValidateResources"
      },
      "ValidateResources": {
        "Type": "Task",
        "Comment": "Validate Resources",
        "Next": "DeployFn",
        "Catch": [
          {
            "Comment": "Try to Release Locks",
            "ErrorEquals": ["BadReleaseError", "PanicError"],
            "ResultPath": "$.error",
            "Next": "ReleaseLockFailureFn"
          }
        ]
      },
      "DeployFn": {
        "Type": "Pass",
        "Result": "Deploy",
        "ResultPath": "$.Task",
        "Next": "Deploy"
      },
      "Deploy": {
        "Type": "Task",
        "Comment": "Create Resources",
        "Next": "WaitForDeploy",
        "Catch": [
          {
            "Comment": "Try to Release Locks and Cleanup any created Resources",
            "ErrorEquals": ["DeployError", "PanicError"],
            "ResultPath": "$.error",
            "Next": "CleanUpFailureFn"
          },
          {
            "Comment": "Try to Release Locks",
            "ErrorEquals": ["HaltError"],
            "ResultPath": "$.error",
            "Next": "ReleaseLockFailureFn"
          }
        ]
      },
      "WaitForDeploy": {
        "Comment": "Give the Deploy time to boot instances",
        "Type": "Wait",
        "Seconds" : 30,
        "Next": "WaitForHealthy"
      },
      "WaitForHealthy": {
        "Type": "Wait",
        "Seconds" : 15,
        "Next": "CheckHealthyFn"
      },
      "CheckHealthyFn": {
        "Type": "Pass",
        "Result": "CheckHealthy",
        "ResultPath": "$.Task",
        "Next": "CheckHealthy"
      },
      "CheckHealthy": {
        "Type": "Task",
        "Comment": "Is the new deploy healthy? Should we continue checking?",
        "Next": "Healthy?",
        "Retry": [ {
          "Comment": "HealthError might occur, just retry a few times",
          "ErrorEquals": ["HealthError", "PanicError"],
          "MaxAttempts": 3,
          "IntervalSeconds": 15
        }],
        "Catch": [{
          "Comment": "HaltError immediately Clean up",
          "ErrorEquals": ["HaltError", "HealthError", "PanicError"],
          "ResultPath": "$.error",
          "Next": "CleanUpFailureFn"
        }]
      },
      "Healthy?": {
        "Comment": "Check the release is $.healthy",
        "Type": "Choice",
        "Choices": [
          {
            "Variable": "$.healthy",
            "BooleanEquals": true,
            "Next": "CleanUpSuccessFn"
          },
          {
            "Variable": "$.healthy",
            "BooleanEquals": false,
            "Next": "WaitForHealthy"
          }
        ],
        "Default": "CleanUpFailureFn"
      },
      "CleanUpSuccessFn": {
        "Type": "Pass",
        "Result": "CleanUpSuccess",
        "ResultPath": "$.Task",
        "Next": "CleanUpSuccess"
      },
      "CleanUpSuccess": {
        "Type": "Task",
        "Comment": "Promote New Resources & Delete Old Resources",
        "Next": "Success",
        "Retry": [ {
          "Comment": "Keep trying to Clean",
          "ErrorEquals": ["CleanUpError", "LockError", "PanicError"],
          "MaxAttempts": 3,
          "IntervalSeconds": 30
        }],
        "Catch": [{
          "ErrorEquals": ["CleanUpError", "LockError", "PanicError"],
          "ResultPath": "$.error",
          "Next": "FailureDirty"
        }]
      },
      "CleanUpFailureFn": {
        "Type": "Pass",
        "Result": "CleanUpFailure",
        "ResultPath": "$.Task",
        "Next": "CleanUpFailure"
      },
      "CleanUpFailure": {
        "Type": "Task",
        "Comment": "Delete New Resources",
        "Next": "ReleaseLockFailureFn",
        "Retry": [ {
          "Comment": "Keep trying to Clean",
          "ErrorEquals": ["CleanUpError", "PanicError"],
          "MaxAttempts": 3,
          "IntervalSeconds": 30
        }],
        "Catch": [{
          "ErrorEquals": ["CleanUpError", "PanicError"],
          "ResultPath": "$.error",
          "Next": "FailureDirty"
        }]
      },
      "ReleaseLockFailureFn": {
        "Type": "Pass",
        "Result": "ReleaseLockFailure",
        "ResultPath": "$.Task",
        "Next": "ReleaseLockFailure"
      },
      "ReleaseLockFailure": {
        "Type": "Task",
        "Comment": "Delete New Resources",
        "Next": "FailureClean",
        "Retry": [ {
          "Comment": "Keep trying to Clean",
          "ErrorEquals": ["LockError", "PanicError"],
          "MaxAttempts": 3,
          "IntervalSeconds": 30
        }],
        "Catch": [{
          "ErrorEquals": ["LockError", "PanicError"],
          "ResultPath": "$.error",
          "Next": "FailureDirty"
        }]
      },
      "FailureClean": {
        "Comment": "Deploy Failed, but no bad resources left behind",
        "Type": "Fail",
        "Error": "FailureClean"
      },
      "FailureDirty": {
        "Comment": "Deploy Failed, Resources left in Bad State, ALERT!",
        "Type": "Fail",
        "Error": "FailureDirty"
      },
      "Success": {
        "Type": "Succeed"
      }
    }
  }`))

	if err != nil {
		return nil, err
	}

	return stateMachine, nil
}

// StateMachineWithTaskHandlers returns
func StateMachineWithTaskHandlers(tfs *handler.TaskFunctions) (*machine.StateMachine, error) {
	stateMachine, err := StateMachine()
	if err != nil {
		return nil, err
	}

	for name, smhandler := range *tfs {
		if err := stateMachine.SetResourceFunction(name, smhandler); err != nil {
			return nil, err
		}

	}

	return stateMachine, nil
}

// TaskFunctions returns
func TaskFunctions() *handler.TaskFunctions {
	return CreateTaskFunctinons(&aws.ClientsStr{})
}

// CreateTaskFunctinons returns
func CreateTaskFunctinons(awsClients aws.Clients) *handler.TaskFunctions {
	tm := handler.TaskFunctions{}
	tm["Validate"] = Validate(awsClients)
	tm["Lock"] = Lock(awsClients)
	tm["ValidateResources"] = ValidateResources(awsClients)
	tm["Deploy"] = Deploy(awsClients)
	tm["CheckHealthy"] = CheckHealthy(awsClients)
	tm["CleanUpSuccess"] = CleanUpSuccess(awsClients)
	tm["CleanUpFailure"] = CleanUpFailure(awsClients)
	tm["ReleaseLockFailure"] = ReleaseLockFailure(awsClients)
	return &tm
}
