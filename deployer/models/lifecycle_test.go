package models

import (
	"testing"

	"github.com/coinbase/step/utils/to"
	"github.com/stretchr/testify/assert"
)

func Test_Lifecycle_Valid(t *testing.T) {
	lc := &LifeCycleHook{
		Transistion: to.Strp("autoscaling:EC2_INSTANCE_LAUNCHING"),
		Role:        to.Strp("role"),
		SNS:         to.Strp("sns"),
	}

	lc.SetDefaults(to.Strp("region"), to.Strp("accountID"), "name")
	assert.NoError(t, lc.ValidateAttributes())
}
