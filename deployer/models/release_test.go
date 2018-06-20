package models

import (
	"testing"

	"github.com/coinbase/step/utils/to"
	"github.com/stretchr/testify/assert"
)

func Test_Release_Validate_Works(t *testing.T) {
	r := MockRelease(t)
	awsc := MockAwsClients(r)
	r.releaseSHA256 = to.SHA256Struct(r)

	MockPrepareRelease(r)

	assert.NoError(t, r.Validate(awsc.S3))
}

func Test_Release_ValidateAttributes_Works(t *testing.T) {
	r := MockRelease(t)
	MockPrepareRelease(r)

	assert.NoError(t, r.ValidateAttributes())
}

func Test_Release_ValidateReleaseSHA_Works(t *testing.T) {
	r := MockRelease(t)
	awsc := MockAwsClients(r)
	r.releaseSHA256 = to.SHA256Struct(r)

	MockPrepareRelease(r)

	assert.NoError(t, r.ValidateReleaseSHA(awsc.S3))
}

func Test_Release_ValidateServices_Works(t *testing.T) {
	r := MockRelease(t)
	MockPrepareRelease(r)

	assert.NoError(t, r.ValidateServices())
}
