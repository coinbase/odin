package deployer

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/coinbase/odin/aws/mocks"
	"github.com/coinbase/odin/deployer/models"
	"github.com/coinbase/step/utils/to"
	"github.com/stretchr/testify/assert"
)

// Test that validate resources fetches the correct resources
func Test_ValidateResources_FetchesCorrectResources(t *testing.T) {
	release := models.MockRelease(t)
	models.MockPrepareRelease(release)

	awsc := models.MockAwsClients(release)
	rel, err := ValidateResources(awsc)(nil, release)
	assert.NoError(t, err)
	res := rel.Services["web"].Resources
	assert.Equal(t, "ami-123456", *res.Image)
	assert.Equal(t, "/odin/project/config/web/web-profile", *res.Profile)
	assert.Equal(t, "project-config-web-old-release", *res.PrevASG)
	assert.Equal(t, []string{"group-id"}, to.StrSlice(res.SecurityGroups))
	assert.Equal(t, []string{"web-elb"}, to.StrSlice(res.ELBs))
	assert.Equal(t, []string{"web-elb-target"}, to.StrSlice(res.TargetGroups))
	assert.Equal(t, []string{"subnet-1"}, to.StrSlice(res.Subnets))
}

// Test that validate resources fails is security group does not exist, or has wrong tags
func Test_ValidateResources_BadSG(t *testing.T) {
	release := models.MockRelease(t)
	models.MockPrepareRelease(release)

	awsc := models.MockAwsClients(release)
	awsc.EC2.AddSecurityGroup("web-sg", *release.ProjectName, *release.ConfigName, "noop", nil)
	_, err := ValidateResources(awsc)(nil, release)
	assert.Error(t, err)
}

// Test that validate resources fails if ELB or target Group has wrong tags
func Test_ValidateResources_BadELB(t *testing.T) {
	release := models.MockRelease(t)
	models.MockPrepareRelease(release)

	awsc := models.MockAwsClients(release)
	awsc.ELB.AddELB("web-elb", *release.ProjectName, *release.ConfigName, "noop")
	_, err := ValidateResources(awsc)(nil, release)
	assert.Error(t, err)
}

// Test that validate resources fails if  IAM role has wrong path
func Test_ValidateResources_BadProfile(t *testing.T) {
	release := models.MockRelease(t)
	models.MockPrepareRelease(release)

	awsc := models.MockAwsClients(release)
	awsc.IAM.AddGetInstanceProfile("web-profile", fmt.Sprintf("/%v/%v/webnoop/", *release.ProjectName, *release.ConfigName))
	_, err := ValidateResources(awsc)(nil, release)
	assert.Error(t, err)
}

func Test_ValidateResources_BadTG(t *testing.T) {
	release := models.MockRelease(t)
	models.MockPrepareRelease(release)

	awsc := models.MockAwsClients(release)
	awsc.ALB.AddTargetGroup(mocks.MockTargetGroup{
		Name:        "web-elb-target",
		ProjectName: *release.ProjectName,
		ConfigName:  *release.ConfigName,
		ServiceName: "noop",
	})
	_, err := ValidateResources(awsc)(nil, release)
	assert.Error(t, err)
}

func Test_ValidateResources_AllowedServiceTg(t *testing.T) {
	release := models.MockRelease(t)
	release.Services["web"].TargetGroups = []*string{to.Strp("other-project-target")}

	models.MockPrepareRelease(release)

	awsc := models.MockAwsClients(release)
	awsc.ALB.AddTargetGroup(mocks.MockTargetGroup{
		Name:           "other-project-target",
		ProjectName:    "other/project",
		ServiceName:    "some-service",
		AllowedService: "project::config::web",
	})
	rel, err := ValidateResources(awsc)(nil, release)
	assert.NoError(t, err)
	res := rel.Services["web"].Resources
	assert.Equal(t, []string{"other-project-target"}, to.StrSlice(res.TargetGroups))
}

// Test Check Healthy
func Test_CheckHealthy_CorrectReport(t *testing.T) {
	release := models.MockRelease(t)
	models.MockPrepareRelease(release)
	release.Services["web"].Resources = &models.ServiceResourceNames{}
	release.Services["web"].CreatedASG = to.Strp("asd")

	awsc := mocks.MockAWS()
	awsc.ASG.AddASG(&autoscaling.Group{Instances: mocks.MakeMockASGInstances(2, 3, 0)})

	assert.Equal(t, false, *release.Healthy)

	res, err := CheckHealthy(awsc)(nil, release)
	assert.NoError(t, err)

	assert.Equal(t, true, *res.Healthy)
	hr := res.Services["web"].HealthReport
	assert.Equal(t, 1, *hr.TargetHealthy)
	assert.Equal(t, 1, *hr.TargetLaunched)
	assert.Equal(t, 2, *hr.Healthy)
	assert.Equal(t, 5, *hr.Launching)
	assert.Equal(t, 0, *hr.Terminating)
}

// Test Check Healthy halts if terming
func Test_CheckHealthy_Terming(t *testing.T) {
	release := models.MockRelease(t)
	models.MockPrepareRelease(release)
	release.Services["web"].Resources = &models.ServiceResourceNames{}
	release.Services["web"].CreatedASG = to.Strp("asd")

	awsc := mocks.MockAWS()
	awsc.ASG.AddASG(&autoscaling.Group{Instances: mocks.MakeMockASGInstances(2, 3, 1)})

	_, err := CheckHealthy(awsc)(nil, release)
	assert.Error(t, err)
}
