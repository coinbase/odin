package models

import (
	"fmt"
	"testing"

	"github.com/coinbase/step/utils/to"
	"github.com/stretchr/testify/assert"
)

func Test_Service_SetGetUserdata(t *testing.T) {
	release := MockMinimalRelease(t)

	service := Service{}
	service.SetUserData(to.Strp("{{RELEASE_ID}}\n{{PROJECT_NAME}}\n{{CONFIG_NAME}}\n{{SERVICE_NAME}}\n"))
	service.SetDefaults(release, "web")

	assert.Equal(t, fmt.Sprintf("%v\n%v\n%v\nweb\n", *release.ReleaseID, *release.ProjectName, *release.ConfigName), *service.UserData())
}

func Test_Service_CreateInput_HealthCheckGracePeriod(t *testing.T) {
	release := MockMinimalRelease(t)
	release.Timeout = to.Intp(10)

	service := Service{}
	service.SetUserData(to.Strp("{{RELEASE_ID}}\n{{PROJECT_NAME}}\n{{CONFIG_NAME}}\n{{SERVICE_NAME}}\n"))
	service.SetDefaults(release, "web")

	input := service.createInput()
	assert.Equal(t, *input.HealthCheckGracePeriod, int64(10))
}
