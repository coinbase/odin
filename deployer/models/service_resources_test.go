package models

import (
	"testing"

	"github.com/coinbase/odin/aws/alb"
	"github.com/coinbase/odin/aws/ami"
	"github.com/coinbase/odin/aws/asg"
	"github.com/coinbase/odin/aws/elb"
	"github.com/coinbase/odin/aws/iam"
	"github.com/coinbase/odin/aws/sg"
	"github.com/coinbase/odin/aws/subnet"
	"github.com/coinbase/step/utils/to"
	"github.com/stretchr/testify/assert"
)

type MockService struct{}

func (*MockService) ProjectName() *string {
	return to.Strp("project")
}

func (*MockService) ConfigName() *string {
	return to.Strp("config")
}

func (*MockService) Name() *string {
	return to.Strp("servicename")
}

func (*MockService) ReleaseID() *string {
	return to.Strp("releaseid")
}

func Test_Service_ValidateImage(t *testing.T) {
	// func ValidateImage(service *Service, im *ami.Image) error {
	assert.Error(t, ValidateImage(&MockService{}, &ami.Image{ImageID: to.Strp("image")}))
	assert.NoError(t, ValidateImage(&MockService{}, &ami.Image{
		ImageID:       to.Strp("image"),
		DeployWithTag: to.Strp("odin"),
	}))
}

func Test_Service_ValidateSubnet(t *testing.T) {
	// func ValidateSubnet(service *Service, subnet *subnet.Subnet) error {
	assert.Error(t, ValidateSubnet(&MockService{}, &subnet.Subnet{SubnetID: to.Strp("subnet")}))

	assert.NoError(t, ValidateSubnet(&MockService{}, &subnet.Subnet{
		SubnetID:      to.Strp("subnet"),
		DeployWithTag: to.Strp("odin"),
	}))
}

func Test_Service_ValidatePrevASG(t *testing.T) {
	// func ValidatePrevASG(service *Service, as *asg.ASG) error {
	assert.Error(t, ValidatePrevASG(&MockService{}, &asg.ASG{}))

	// Project Name
	assert.Error(t, ValidatePrevASG(&MockService{}, &asg.ASG{
		ProjectNameTag: to.Strp("notproject"),
		ConfigNameTag:  to.Strp("config"),
		ServiceNameTag: to.Strp("servicename"),
		ReleaseIDTag:   to.Strp("new_releaseid"),
	}))

	// Config Name
	assert.Error(t, ValidatePrevASG(&MockService{}, &asg.ASG{
		ProjectNameTag: to.Strp("project"),
		ConfigNameTag:  to.Strp("notconfig"),
		ServiceNameTag: to.Strp("servicename"),
		ReleaseIDTag:   to.Strp("new_releaseid"),
	}))
	// Service Name
	assert.Error(t, ValidatePrevASG(&MockService{}, &asg.ASG{
		ProjectNameTag: to.Strp("project"),
		ConfigNameTag:  to.Strp("config"),
		ServiceNameTag: to.Strp("notservicename"),
		ReleaseIDTag:   to.Strp("new_releaseid"),
	}))
	// ReleaseID the same
	assert.Error(t, ValidatePrevASG(&MockService{}, &asg.ASG{
		ProjectNameTag: to.Strp("project"),
		ConfigNameTag:  to.Strp("config"),
		ServiceNameTag: to.Strp("servicename"),
		ReleaseIDTag:   to.Strp("releaseid"),
	}))

	assert.NoError(t, ValidatePrevASG(&MockService{}, &asg.ASG{
		ProjectNameTag: to.Strp("project"),
		ConfigNameTag:  to.Strp("config"),
		ServiceNameTag: to.Strp("servicename"),
		ReleaseIDTag:   to.Strp("new_releaseid"),
	}))
}

func Test_Service_ValidateIAMProfile(t *testing.T) {
	// func ValidateIAMProfile(service *Service, profile *iam.Profile) error {
	assert.Error(t, ValidateIAMProfile(&MockService{}, &iam.Profile{}))

	assert.NoError(t, ValidateIAMProfile(&MockService{}, &iam.Profile{
		Path: to.Strp("/project/config/servicename/"),
	}))
}

func Test_Service_ValidateSecurityGroup(t *testing.T) {
	// func ValidateSecurityGroup(service *Service, sc *sg.SecurityGroup) error {
	assert.Error(t, ValidateSecurityGroup(&MockService{}, &sg.SecurityGroup{}))

	// Project Name
	assert.Error(t, ValidateSecurityGroup(&MockService{}, &sg.SecurityGroup{
		ProjectNameTag: to.Strp("notproject"),
		ConfigNameTag:  to.Strp("config"),
		ServiceNameTag: to.Strp("servicename"),
	}))

	// Config Name
	assert.Error(t, ValidateSecurityGroup(&MockService{}, &sg.SecurityGroup{
		ProjectNameTag: to.Strp("project"),
		ConfigNameTag:  to.Strp("notconfig"),
		ServiceNameTag: to.Strp("servicename"),
	}))

	// Service Name
	assert.Error(t, ValidateSecurityGroup(&MockService{}, &sg.SecurityGroup{
		ProjectNameTag: to.Strp("project"),
		ConfigNameTag:  to.Strp("config"),
		ServiceNameTag: to.Strp("notservicename"),
	}))

	assert.NoError(t, ValidateSecurityGroup(&MockService{}, &sg.SecurityGroup{
		ProjectNameTag: to.Strp("project"),
		ConfigNameTag:  to.Strp("config"),
		ServiceNameTag: to.Strp("servicename"),
	}))

	assert.NoError(t, ValidateSecurityGroup(&MockService{}, &sg.SecurityGroup{
		ProjectNameTag: to.Strp("_all"),
		ConfigNameTag:  to.Strp("config"),
		ServiceNameTag: to.Strp("servicename"),
	}))

	assert.NoError(t, ValidateSecurityGroup(&MockService{}, &sg.SecurityGroup{
		ProjectNameTag: to.Strp("_all"),
		ConfigNameTag:  to.Strp("config"),
		ServiceNameTag: to.Strp("servicename"),
	}))

	assert.NoError(t, ValidateSecurityGroup(&MockService{}, &sg.SecurityGroup{
		ProjectNameTag: to.Strp("_all"),
		ConfigNameTag:  to.Strp("_all"),
		ServiceNameTag: to.Strp("_all"),
	}))
}

func Test_Service_ValidateELB(t *testing.T) {
	// func ValidateELB(service *Service, lb *elb.LoadBalancer) error {
	assert.Error(t, ValidateELB(&MockService{}, &elb.LoadBalancer{}))

	// Project Name
	assert.Error(t, ValidateELB(&MockService{}, &elb.LoadBalancer{
		ProjectNameTag: to.Strp("notproject"),
		ConfigNameTag:  to.Strp("config"),
	}))

	// Config Name
	assert.Error(t, ValidateELB(&MockService{}, &elb.LoadBalancer{
		ProjectNameTag: to.Strp("project"),
		ConfigNameTag:  to.Strp("notconfig"),
	}))

	assert.NoError(t, ValidateELB(&MockService{}, &elb.LoadBalancer{
		ProjectNameTag: to.Strp("project"),
		ConfigNameTag:  to.Strp("config"),
		ServiceNameTag: to.Strp("servicename"),
	}))
}

func Test_Service_ValidateTargetGroup(t *testing.T) {
	// func ValidateTargetGroup(service *Service, tg *alb.TargetGroup) error {
	assert.Error(t, ValidateTargetGroup(&MockService{}, &alb.TargetGroup{}))

	// Project Name
	assert.Error(t, ValidateTargetGroup(&MockService{}, &alb.TargetGroup{
		ProjectNameTag: to.Strp("notproject"),
		ConfigNameTag:  to.Strp("config"),
		ServiceNameTag: to.Strp("servicename"),
	}))

	// Config Name
	assert.Error(t, ValidateTargetGroup(&MockService{}, &alb.TargetGroup{
		ProjectNameTag: to.Strp("project"),
		ConfigNameTag:  to.Strp("notconfig"),
		ServiceNameTag: to.Strp("servicename"),
	}))

	// Service Name
	assert.Error(t, ValidateTargetGroup(&MockService{}, &alb.TargetGroup{
		ProjectNameTag: to.Strp("project"),
		ConfigNameTag:  to.Strp("config"),
		ServiceNameTag: to.Strp("notservicename"),
	}))

	assert.NoError(t, ValidateTargetGroup(&MockService{}, &alb.TargetGroup{
		ProjectNameTag: to.Strp("project"),
		ConfigNameTag:  to.Strp("config"),
		ServiceNameTag: to.Strp("servicename"),
	}))
}
