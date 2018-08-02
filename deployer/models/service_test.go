package models

import (
	"fmt"
	"testing"
	"time"

	"github.com/coinbase/step/bifrost"
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

func Test_Service_ServiceID(t *testing.T) {
	release := &Release{Release: bifrost.Release{ProjectName: to.Strp("project"), ConfigName: to.Strp("config"), CreatedAt: &time.Time{}}}
	service := &Service{
		release:     release,
		ServiceName: to.Strp("service"),
	}

	assert.Equal(t, *service.ServiceID(), "project-config-0001-01-01T00-00-00Z-service")

	service.ServiceName = to.Strp("this_will_cause_a_name_longer_than_80_characters")
	assert.Equal(t, *service.ServiceID(), "project-co-0001-01-01T00-00-00Z-this_will_cause_a_name_longer_than_80_characters")
	assert.Equal(t, len(*service.ServiceID()), 80)

	// This should not happen due to Char limit on ServiceName, but still should not error
	service.ServiceName = to.Strp("this_will_cause_a_name_longer_than_80_characters_which_is_longer_than_the_max_crazzy")
	assert.Equal(t, *service.ServiceID(), "project-config-0001-01-01T00-00-00Z-this_will_cause_a_name_longer_than_80_charac")
	assert.Equal(t, len(*service.ServiceID()), 80)

}
