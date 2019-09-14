package models

import (
	"testing"

	"github.com/coinbase/odin/aws/asg"
	"github.com/coinbase/step/utils/to"
	"github.com/stretchr/testify/assert"
)

func Test_Release_ValidateSafeDeploy_Works(t *testing.T) {
	release := MockRelease(t)
	MockPrepareRelease(release)

	awsc := MockAwsClients(release)

	previousRelease := MockRelease(t)
	previousRelease.ReleaseID = to.Strp("prevReleaseID")

	// Add release to S3 Mock
	addReleaseS3Objects(awsc, previousRelease)

	err := release.ValidateSafeDeploy(awsc.S3, &ReleaseResources{
		PreviousReleaseID: previousRelease.ReleaseID,
		PreviousASGs:      map[string]*asg.ASG{"a": nil},
		ServiceResources:  map[string]*ServiceResources{"a": nil},
	})

	assert.NoError(t, err)
}

func Test_Release_validateSafeDeploy_Works(t *testing.T) {
	release := MockRelease(t)
	previousRelease := MockRelease(t)

	err := release.validateSafeDeploy(previousRelease)
	assert.NoError(t, err)
}

func Test_Release_validateSafeDeploy_SubnetErrors(t *testing.T) {
	release := MockRelease(t)
	release.Subnets = []*string{to.Strp("not")}

	previousRelease := MockRelease(t)

	err := release.validateSafeDeploy(previousRelease)
	assert.Error(t, err)
	assert.Regexp(t, "Subnet", err.Error())
}

func Test_Release_validateSafeDeploy_ImageErrors(t *testing.T) {
	release := MockRelease(t)
	release.Image = to.Strp("not_image")

	previousRelease := MockRelease(t)

	err := release.validateSafeDeploy(previousRelease)
	assert.Error(t, err)
	assert.Regexp(t, "Image", err.Error())
}
