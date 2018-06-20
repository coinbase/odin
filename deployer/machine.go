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
    "StartAt": "Validate",
    "States": {
      "Validate": {
        "Type": "TaskFn",
        "Comment": "Validate and Set Defaults",
        "Next": "Lock",
        "Catch": [
          {
            "Comment": "Bad Input, straight to Failure Clean, dont pass go dont collect $200",
            "ErrorEquals": ["BadReleaseError", "PanicError", "UnmarshalError"],
            "ResultPath": "$.error",
            "Next": "FailureClean"
          }
        ]
      },
      "Lock": {
        "Type": "TaskFn",
        "Comment": "Grab Lock",
        "Next": "ValidateResources",
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
            "Next": "ReleaseLockFailure"
          },
          {
            "Comment": "Panic is not good",
            "ErrorEquals": ["PanicError"],
            "ResultPath": "$.error",
            "Next": "FailureDirty"
          }
        ]
      },
      "ValidateResources": {
        "Type": "TaskFn",
        "Comment": "Validate Resources",
        "Next": "Deploy",
        "Catch": [
          {
            "Comment": "Try to Release Locks",
            "ErrorEquals": ["BadReleaseError", "PanicError"],
            "ResultPath": "$.error",
            "Next": "ReleaseLockFailure"
          }
        ]
      },
      "Deploy": {
        "Type": "TaskFn",
        "Comment": "Create Resources",
        "Next": "WaitForDeploy",
        "Catch": [
          {
            "Comment": "Try to Release Locks and Cleanup any created Resources",
            "ErrorEquals": ["DeployError", "PanicError"],
            "ResultPath": "$.error",
            "Next": "CleanUpFailure"
          },
          {
            "Comment": "Try to Release Locks",
            "ErrorEquals": ["HaltError", "BadReleaseError"],
            "ResultPath": "$.error",
            "Next": "ReleaseLockFailure"
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
        "Next": "CheckHealthy"
      },
      "CheckHealthy": {
        "Type": "TaskFn",
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
          "Next": "CleanUpFailure"
        }]
      },
      "Healthy?": {
        "Comment": "Check the release is $.healthy",
        "Type": "Choice",
        "Choices": [
          {
            "Variable": "$.healthy",
            "BooleanEquals": true,
            "Next": "CleanUpSuccess"
          },
          {
            "Variable": "$.healthy",
            "BooleanEquals": false,
            "Next": "WaitForHealthy"
          }
        ],
        "Default": "CleanUpFailure"
      },
      "CleanUpSuccess": {
        "Type": "TaskFn",
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
      "CleanUpFailure": {
        "Type": "TaskFn",
        "Comment": "Delete New Resources",
        "Next": "ReleaseLockFailure",
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
      "ReleaseLockFailure": {
        "Type": "TaskFn",
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
