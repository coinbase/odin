package models

import (
	"testing"

	"github.com/coinbase/step/utils/to"
	"github.com/stretchr/testify/assert"
)

func Test_Autoscaling_Valid(t *testing.T) {
	asg := &AutoScalingConfig{}
	asg.SetDefaults(nil, nil)
	assert.NoError(t, asg.ValidateAttributes())
}

func Test_PolicyNames_Uniq(t *testing.T) {
	asg := &AutoScalingConfig{
		Policies: []*Policy{
			&Policy{Type: to.Strp("cpu_scale_down")},
			&Policy{Type: to.Strp("cpu_scale_up")},
		},
	}
	asg.SetDefaults(to.Strp("service_id"), nil)
	assert.NoError(t, asg.ValidateAttributes())

	asg.Policies[0].Type = to.Strp("cpu_scale_up")
	assert.Error(t, asg.ValidateAttributes())

	asg.Policies[0].NameVal = to.Strp("override_name")
	assert.NoError(t, asg.ValidateAttributes())
}

func Test_Autoscaling_HealthCheckGracePeriod(t *testing.T) {
	asg := &AutoScalingConfig{}
	assert.Nil(t, asg.HealthCheckGracePeriod)

	// Default to timeout
	asg.SetDefaults(nil, to.Intp(10))
	assert.Equal(t, *asg.HealthCheckGracePeriod, int64(10))

	// Min to timeout
	asg.HealthCheckGracePeriod = to.Int64p(100)
	asg.SetDefaults(nil, to.Intp(20))
	assert.Equal(t, *asg.HealthCheckGracePeriod, int64(20))

	// min to HealthCheck
	asg.HealthCheckGracePeriod = to.Int64p(100)
	asg.SetDefaults(nil, to.Intp(2000))
	assert.Equal(t, *asg.HealthCheckGracePeriod, int64(100))
}
