package client

import (
	"encoding/json"
	"testing"

	"github.com/coinbase/odin/deployer/models"
	"github.com/coinbase/step/execution"
	"github.com/coinbase/step/utils/to"
	"github.com/stretchr/testify/assert"
)

func minimalRelease(t *testing.T) *models.Release {
	var r models.Release
	err := json.Unmarshal([]byte(`
  {
    "release_id": "rr",
    "project_name": "project",
    "config_name": "config",
    "ami": "ami-123456",
    "subnets": ["subnet-1"],
    "user_data": "echo DATE",
    "services": {
      "web": {
        "instance_type": "t2.small",
        "security_groups": ["web-sg"]
      }
    }
  }
  `), &r)

	assert.NoError(t, err)
	return &r
}

func Test_releaseFromFileOrJSON(t *testing.T) {
	release, err := releaseFromFileOrJSON(to.Strp(` {
    "release_id": "rr",
    "project_name": "project",
    "config_name": "config",
    "ami": "ami-123456",
    "subnets": ["subnet-1"],
    "user_data": "echo DATE",
    "services": {
      "web": {
        "instance_type": "t2.small",
        "security_groups": ["web-sg"]
      }
    }
  }`), to.Strp("region"), to.Strp("account"))
	assert.NoError(t, err)

	assert.Equal(t, "rr", *release.ReleaseID)
	assert.Equal(t, "project", *release.ProjectName)
}

func Test_releaseFromFileOrJSON_badRelease(t *testing.T) {
	_, err := releaseFromFileOrJSON(to.Strp(`{}`), to.Strp("region"), to.Strp("account"))
	assert.Error(t, err)
}

func Test_releaseFromFileOrJSON_badJSON(t *testing.T) {
	_, err := releaseFromFileOrJSON(to.Strp(`{`), to.Strp("region"), to.Strp("account"))
	assert.Error(t, err)
}

func Test_releaseFromFileOrJSON_UnknownKey(t *testing.T) {
	_, err := releaseFromFileOrJSON(to.Strp(`{"bad_key": "val"}`), to.Strp("region"), to.Strp("account"))
	assert.Error(t, err)
}

func createStateDetails(release *models.Release, tn string) *execution.StateDetails {
	lo, _ := to.PrettyJSON((release))
	return &execution.StateDetails{
		LastOutput:   &lo,
		LastTaskName: &tn,
	}
}

func waiterStrTest(t *testing.T, r *models.Release) string {
	spinnerCounter = 5
	str, err := waiterStr(to.Strp("RUNNING"), createStateDetails(r, "TaskName"))
	assert.NoError(t, err)
	return str
}

func Test_waiterStr(t *testing.T) {
	r := minimalRelease(t)
	assert.Equal(t, "-RUNNING(TaskName)", waiterStrTest(t, r))

	r.Services["web"].HealthReport = &models.HealthReport{
		TargetHealthy:  to.Intp(3),
		TargetLaunched: to.Intp(5),
		Healthy:        to.Intp(1),
		Launching:      to.Intp(5),
		Terminating:    to.Intp(0),
	}

	waiterStrTest(t, r) // Checks errors
}
