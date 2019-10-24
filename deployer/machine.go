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
        "Resource": "arn:aws:lambda:{{aws_region}}:{{aws_account}}:function:{{lambda_name}}",
        "Comment": "Validate and Set Defaults",
        "Next": "Lock",
        "Catch": [
          {
            "Comment": "Bad Input, straight to Failure Clean, dont pass go dont collect $200",
            "ErrorEquals": ["States.ALL"],
            "ResultPath": "$.error",
            "Next": "FailureClean"
          }
        ]
      },
      "Lock": {
        "Type": "TaskFn",
        "Resource": "arn:aws:lambda:{{aws_region}}:{{aws_account}}:function:{{lambda_name}}",
        "Comment": "Grab Lock",
        "Next": "ValidateResources",
        "Catch": [
          {
            "Comment": "Bad Input, straight to Failure Clean",
            "ErrorEquals": ["LockExistsError"],
            "ResultPath": "$.error",
            "Next": "FailureClean"
          },
          {
            "Comment": "Release Lock if you created it",
            "ErrorEquals": ["States.ALL"],
            "ResultPath": "$.error",
            "Next": "ReleaseLockFailure"
          }
        ]
      },
      "ValidateResources": {
        "Type": "TaskFn",
        "Resource": "arn:aws:lambda:{{aws_region}}:{{aws_account}}:function:{{lambda_name}}",
        "Comment": "Validate Resources",
        "Next": "Deploy",
        "Catch": [
          {
            "Comment": "Try to Release Locks",
            "ErrorEquals": ["States.ALL"],
            "ResultPath": "$.error",
            "Next": "ReleaseLockFailure"
          }
        ]
      },
      "Deploy": {
        "Type": "TaskFn",
        "Resource": "arn:aws:lambda:{{aws_region}}:{{aws_account}}:function:{{lambda_name}}",
        "Comment": "Create Resources",
        "Next": "WaitForDeploy",
        "Catch": [
          {
            "Comment": "Try to Release Locks",
            "ErrorEquals": ["HaltError"],
            "ResultPath": "$.error",
            "Next": "ReleaseLockFailure"
          },
          {
            "Comment": "Try to Release Locks and Cleanup any created Resources",
            "ErrorEquals": ["States.ALL"],
            "ResultPath": "$.error",
            "Next": "DetachForFailure"
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
        "SecondsPath" : "$.wait_for_healthy",
        "Next": "CheckHealthy"
      },
      "CheckHealthy": {
        "Type": "TaskFn",
        "Resource": "arn:aws:lambda:{{aws_region}}:{{aws_account}}:function:{{lambda_name}}",
        "Comment": "Is the new deploy healthy? Should we continue checking?",
        "Next": "Healthy?",
        "Retry": [{
          "Comment": "Do not retry on HaltError",
          "ErrorEquals": ["HaltError"],
          "MaxAttempts": 0
        },
        {
          "Comment": "HealthError might occur, just retry a few times",
          "ErrorEquals": ["States.ALL"],
          "MaxAttempts": 3,
          "IntervalSeconds": 15
        }],
        "Catch": [{
          "Comment": "HaltError immediately Clean up",
          "ErrorEquals": ["States.ALL"],
          "ResultPath": "$.error",
          "Next": "DetachForFailure"
        }]
      },
      "Healthy?": {
        "Comment": "Check the release is $.healthy",
        "Type": "Choice",
        "Choices": [
          {
            "Variable": "$.healthy",
            "BooleanEquals": true,
            "Next": "DetachForSuccess"
          },
          {
            "Variable": "$.healthy",
            "BooleanEquals": false,
            "Next": "WaitForHealthy"
          }
        ],
        "Default": "DetachForFailure"
      },
      "DetachForSuccess": {
        "Type": "TaskFn",
        "Resource": "arn:aws:lambda:{{aws_region}}:{{aws_account}}:function:{{lambda_name}}",
        "Comment": "Detach Old ASGs",
        "Next": "WaitDetachForSuccess",
        "Retry": [{
          "Comment": "Retry on Detach Error, for 10 minutes",
          "ErrorEquals": ["DetachError"],
          "MaxAttempts": 60,
          "IntervalSeconds": 10
         },{
          "Comment": "Keep trying to Clean",
          "ErrorEquals": ["States.ALL"],
          "MaxAttempts": 3,
          "IntervalSeconds": 60
        }],
        "Catch": [{
          "Comment": "Force the deletion rather than fail",
          "ErrorEquals": ["States.ALL"],
          "ResultPath": "$.error",
          "Next": "WaitDetachForSuccess"
        }]
      },
      "WaitDetachForSuccess": {
        "Comment": "Give detach a little time to do what it does",
        "Type": "Wait",
        "SecondsPath" : "$.wait_for_detach",
        "Next": "CleanUpSuccess"
      },
      "CleanUpSuccess": {
        "Type": "TaskFn",
        "Resource": "arn:aws:lambda:{{aws_region}}:{{aws_account}}:function:{{lambda_name}}",
        "Comment": "Promote New Resources & Delete Old Resources",
        "Next": "Success",
        "Retry": [{
          "Comment": "Keep trying to Clean",
          "ErrorEquals": ["States.ALL"],
          "MaxAttempts": 3,
          "IntervalSeconds": 60
        }],
        "Catch": [{
          "ErrorEquals": ["States.ALL"],
          "ResultPath": "$.error",
          "Next": "FailureDirty"
        }]
      },
      "DetachForFailure": {
        "Type": "TaskFn",
        "Resource": "arn:aws:lambda:{{aws_region}}:{{aws_account}}:function:{{lambda_name}}",
        "Comment": "Detach Old ASGs",
        "Next": "WaitDetachForFailure",
        "Retry": [{
          "Comment": "Retry on Detach Error",
          "ErrorEquals": ["DetachError"],
          "MaxAttempts": 10,
          "IntervalSeconds": 30
         },{
          "Comment": "Keep trying to Clean",
          "ErrorEquals": ["States.ALL"],
          "MaxAttempts": 3,
          "IntervalSeconds": 60
        }],
        "Catch": [{
          "Comment": "Force the deletion rather than fail",
          "ErrorEquals": ["States.ALL"],
          "ResultPath": "$.error",
          "Next": "WaitDetachForFailure"
        }]
      },
      "WaitDetachForFailure": {
        "Comment": "Give detach a little time to do what it does",
        "Type": "Wait",
        "Seconds" : 10,
        "Next": "CleanUpFailure"
      },
      "CleanUpFailure": {
        "Type": "TaskFn",
        "Resource": "arn:aws:lambda:{{aws_region}}:{{aws_account}}:function:{{lambda_name}}",
        "Comment": "Delete New Resources",
        "Next": "ReleaseLockFailure",
        "Retry": [{
          "Comment": "Keep trying to Clean",
          "ErrorEquals": ["States.ALL"],
          "MaxAttempts": 3,
          "IntervalSeconds": 30
        }],
        "Catch": [{
          "ErrorEquals": ["States.ALL"],
          "ResultPath": "$.error",
          "Next": "FailureDirty"
        }]
      },
      "ReleaseLockFailure": {
        "Type": "TaskFn",
        "Resource": "arn:aws:lambda:{{aws_region}}:{{aws_account}}:function:{{lambda_name}}",
        "Comment": "Delete New Resources",
        "Next": "FailureClean",
        "Retry": [ {
          "Comment": "Keep trying to Clean",
          "ErrorEquals": ["States.ALL"],
          "MaxAttempts": 3,
          "IntervalSeconds": 30
        }],
        "Catch": [{
          "ErrorEquals": ["States.ALL"],
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

// TaskHandlers returns
func TaskHandlers() *handler.TaskHandlers {
  return CreateTaskFunctinons(&aws.ClientsStr{})
}

// CreateTaskFunctinons returns
func CreateTaskFunctinons(awsc aws.Clients) *handler.TaskHandlers {
  tm := handler.TaskHandlers{}
  tm["Validate"] = Validate(awsc)
  tm["Lock"] = Lock(awsc)
  tm["ValidateResources"] = ValidateResources(awsc)
  tm["Deploy"] = Deploy(awsc)
  tm["CheckHealthy"] = CheckHealthy(awsc)

  // success
  tm["DetachForSuccess"] = DetachForSuccess(awsc)
  tm["CleanUpSuccess"] = CleanUpSuccess(awsc)

  // Failure
  tm["DetachForFailure"] = DetachForFailure(awsc)
  tm["CleanUpFailure"] = CleanUpFailure(awsc)
  tm["ReleaseLockFailure"] = ReleaseLockFailure(awsc)
  return &tm
}
