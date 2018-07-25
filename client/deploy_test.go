package client

import (
	"testing"

	"github.com/coinbase/odin/aws/mocks"
	"github.com/coinbase/step/utils/to"
	"github.com/stretchr/testify/assert"
)

func Test_Deploy(t *testing.T) {
	awsc := mocks.MockAWS()
	r := minimalRelease(t)
	r.Release.SetDefaults(to.Strp("region"), to.Strp("accountid"), "")
	r.SetUserData(to.Strp("#cloud_config"))

	err := deploy(awsc, r, to.Strp("deployerARN"))
	assert.NoError(t, err)
}
