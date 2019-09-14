package models

import (
  "encoding/json"
  "fmt"
  "testing"
  "time"

  "github.com/coinbase/odin/aws/mocks"
  "github.com/coinbase/step/utils/to"
  "github.com/stretchr/testify/assert"
)

//////////
// Mock AWS Clients
//////////

// MockPrepareRelease mocks
func MockPrepareRelease(release *Release) {
  release.Release.SetDefaults(to.Strp("region"), to.Strp("account"), "")
  release.SetDefaults()
  if release.UserData() == nil {
    release.SetUserData(to.Strp("#cloud_config"))
  }

  release.UserDataSHA256 = to.Strp(to.SHA256Str(release.UserData()))
}

// MockAwsClients mocks
func MockAwsClients(release *Release) *mocks.MockClients {
  awsc := mocks.MockAWS()

  if release.ProjectName != nil && release.ConfigName != nil {
    awsc.ASG.AddPreviousRuntimeResources(*release.ProjectName, *release.ConfigName, "web", "old-release")

    awsc.EC2.AddSecurityGroup("web-sg", *release.ProjectName, *release.ConfigName, "web", nil)
    awsc.EC2.AddImage("ubuntu", "ami-123456")
    awsc.EC2.AddSubnet("private-subnet", "subnet-1")

    awsc.ELB.AddELB("web-elb", *release.ProjectName, *release.ConfigName, "web")
    awsc.ALB.AddTargetGroup(mocks.MockTargetGroup{
      Name:        "web-elb-target",
      ProjectName: *release.ProjectName,
      ConfigName:  *release.ConfigName,
      ServiceName: "web",
    })

    awsc.IAM.AddGetInstanceProfile("web-profile", fmt.Sprintf("/odin/%v/%v/web/", *release.ProjectName, *release.ConfigName))
    awsc.IAM.AddGetRole("sns_role")

    // Upload items to S3
    if release.ReleaseID == nil {
      release.ReleaseID = to.Strp("rr")
    }

    addReleaseS3Objects(awsc, release)
  }

  return awsc
}

func addReleaseS3Objects(awsc *mocks.MockClients, release *Release) {
  if release.UserData() == nil {
    release.SetUserData(to.Strp("#cloud_config"))
  }

  awsc.S3.AddGetObject(*release.UserDataPath(), *release.UserData(), nil)
  release.UserDataSHA256 = to.Strp(to.SHA256Str(release.UserData()))

  raw, _ := json.Marshal(release)
  awsc.S3.AddGetObject(*release.ReleasePath(), string(raw), nil)
}

//////////
// MockObjects
//////////

// MockMinimalRelease mocks
func MockMinimalRelease(t *testing.T) *Release {
  var r Release
  err := json.Unmarshal([]byte(`
  {
    "aws_account_id": "000000",
    "release_id": "rr",
    "project_name": "project",
    "config_name": "config",
    "ami": "ami-123456",
    "subnets": ["subnet-1"],
    "services": {
      "web": {
        "instance_type": "t2.small",
        "security_groups": ["web-sg"]
      }
    }
  }
  `), &r)

  assert.NoError(t, err)
  r.CreatedAt = to.Timep(time.Now())

  return &r
}

// MockRelease mocks
func MockRelease(t *testing.T) *Release {
  var r Release
  err := json.Unmarshal([]byte(`
  {
    "aws_account_id": "000000",
    "release_id": "1",
    "project_name": "project",
    "config_name": "config",
    "bucket": "bucket",
    "ami": "ubuntu",
    "subnets": ["private-subnet"],
    "timeout": 1,
    "lifecycle": {
      "TermHook" : {
        "transition": "autoscaling:EC2_INSTANCE_TERMINATING",
        "role": "sns_role",
        "sns": "target",
        "heartbeat_timeout": 300
      }
    },
    "services": {
      "web": {
        "instance_type": "t2.small",
        "security_groups": ["web-sg"],
        "elbs": ["web-elb"],
        "target_groups": ["web-elb-target"],
        "profile" : "web-profile",
        "ebs_volume_size": 120,
        "tags": {
          "custom": "tag"
        },
        "autoscaling": {
          "min_size": 1,
          "max_size": 1,
          "max_terms": 0,
          "spread": 0.5,
          "default_cooldown": 10,
          "health_check_grace_period": 10,
          "policies": [
            {
              "name": "asd",
              "type": "cpu_scale_up",
              "scaling_adjustment": 5,
              "threshold" : 25,
              "period": 2,
              "evaluation_periods": 10
            },
            {
              "type": "cpu_scale_down",
              "scaling_adjustment": -1,
              "threshold" : 15
            }
          ]
        }
      }
    }
  }
  `), &r)

  assert.NoError(t, err)

  r.CreatedAt = to.Timep(time.Now())

  return &r
}
