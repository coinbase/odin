package models

import (
	"testing"

	"github.com/coinbase/odin/aws/asg"
	"github.com/coinbase/step/utils/to"
	"github.com/stretchr/testify/assert"
)

func Test_Release_ValidateSafeRelease_Works(t *testing.T) {
	release := MockRelease(t)
	MockPrepareRelease(release)

	awsc := MockAwsClients(release)

	previousRelease := MockRelease(t)
	previousRelease.ReleaseID = to.Strp("prevReleaseID")

	// Add release to S3 Mock
	AddReleaseS3Objects(awsc, previousRelease)

	err := release.ValidateSafeRelease(awsc.S3, &ReleaseResources{
		PreviousReleaseID: previousRelease.ReleaseID,
		PreviousASGs:      map[string]*asg.ASG{"a": nil},
		ServiceResources:  map[string]*ServiceResources{"a": nil},
	})

	assert.NoError(t, err)
}

func Test_Release_validateSafeRelease_Works(t *testing.T) {
	release := MockRelease(t)
	previousRelease := MockRelease(t)

	err := release.validateSafeRelease(previousRelease)
	assert.NoError(t, err)
}

func Test_Release_validateSafeRelease_Subnet_Image(t *testing.T) {
	// Subnet
	release := MockRelease(t)
	release.Subnets = []*string{to.Strp("not")}

	validateSafeErrorTest(t, release, "Subnet")

	// Image
	release = MockRelease(t)
	release.Image = to.Strp("not_image")

	validateSafeErrorTest(t, release, "Image")
}

func Test_Release_validateSafeRelease_Service(t *testing.T) {
	// ELB
	release := MockRelease(t)
	release.Services["web"].ELBs = []*string{to.Strp("not")}

	validateSafeErrorTest(t, release, "ELB")

	// TargetGroup
	release = MockRelease(t)
	release.Services["web"].TargetGroups = []*string{to.Strp("not")}

	validateSafeErrorTest(t, release, "TargetGroup")

	//Instance Type
	release = MockRelease(t)
	release.Services["web"].InstanceType = to.Strp("not")

	validateSafeErrorTest(t, release, "InstanceType")

	// Security Group
	release = MockRelease(t)
	release.Services["web"].SecurityGroups = []*string{to.Strp("not")}

	validateSafeErrorTest(t, release, "SecurityGroup")

	// Profile
	release = MockRelease(t)
	release.Services["web"].Profile = to.Strp("not")

	validateSafeErrorTest(t, release, "Profile")
}

func Test_Release_validateSafeRelease_Autoscaling(t *testing.T) {
	// ELB
	release := MockRelease(t)
	release.Services["web"].Autoscaling.MinSize = to.Int64p(64)

	validateSafeErrorTest(t, release, "MinSize")
}

// Test Util
func validateSafeErrorTest(t *testing.T, release *Release, errStr string) {
	previousRelease := MockRelease(t)

	err := release.validateSafeRelease(previousRelease)
	assert.Error(t, err)
	if err != nil {
		assert.Regexp(t, errStr, err.Error())
	}
}
