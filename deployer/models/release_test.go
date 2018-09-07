package models

import (
	"testing"

	"github.com/coinbase/step/utils/to"
	"github.com/stretchr/testify/assert"
)

func Test_Release_Validate_Works(t *testing.T) {
	r := MockRelease(t)
	awsc := MockAwsClients(r)
	r.ReleaseSHA256 = to.SHA256Struct(r)

	MockPrepareRelease(r)

	assert.NoError(t, r.Validate(awsc.S3))
}

func Test_Release_ValidateServices_Works(t *testing.T) {
	r := MockRelease(t)
	MockPrepareRelease(r)

	assert.NoError(t, r.ValidateServices())
}

func Test_SetDefaults_Sets_WaitForHealthy(t *testing.T) {
	r := MockRelease(t)
	MockPrepareRelease(r)

	assert.Equal(t, 15, *r.WaitForHealthy)

	r = MockRelease(t)
	r.Timeout = to.Intp(3600)
	MockPrepareRelease(r)
	assert.Equal(t, 60, *r.WaitForHealthy)

	r = MockRelease(t)
	r.Timeout = to.Intp(8000)
	MockPrepareRelease(r)
	assert.Equal(t, 120, *r.WaitForHealthy)
}
