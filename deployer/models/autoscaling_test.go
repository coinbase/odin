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

func Test_DesiredCapacity(t *testing.T) {
	assert.Equal(t, 1, desiredCapacity(1, 3, -1))
	assert.Equal(t, 2, desiredCapacity(1, 3, 2))
	assert.Equal(t, 2, desiredCapacity(1, 3, 2))

	asg := &AutoScalingConfig{MinSize: to.Int64p(1), MaxSize: to.Int64p(3)}
	assert.Equal(t, 1, asg.DesiredCapacity(to.Int64p(1)))
	assert.Equal(t, 2, asg.DesiredCapacity(to.Int64p(2)))
	assert.Equal(t, 3, asg.DesiredCapacity(to.Int64p(3)))
}

func Test_TargetCapacity(t *testing.T) {
	assert.Equal(t, 1, targetCapacity(1, 1, 0))
	assert.Equal(t, 3, targetCapacity(3, 2, 1))

	asg := &AutoScalingConfig{MinSize: to.Int64p(1), MaxSize: to.Int64p(10), Spread: to.Float64p(0.5)}
	assert.Equal(t, 1, asg.TargetCapacity(nil))
	assert.Equal(t, 3, asg.TargetCapacity(to.Int64p(2)))
	assert.Equal(t, 6, asg.TargetCapacity(to.Int64p(4)))
	assert.Equal(t, 9, asg.TargetCapacity(to.Int64p(6)))
	assert.Equal(t, 10, asg.TargetCapacity(to.Int64p(8)))
	assert.Equal(t, 10, asg.TargetCapacity(to.Int64p(10)))
}

func Test_TargetHealthy(t *testing.T) {
	assert.Equal(t, 1, targetHealthy(1, 1, 0))
	assert.Equal(t, 1, targetHealthy(1, 3, 1))

	asg := &AutoScalingConfig{MinSize: to.Int64p(1), MaxSize: to.Int64p(10), Spread: to.Float64p(0.5)}
	assert.Equal(t, 1, asg.TargetHealthy(nil))
	assert.Equal(t, 1, asg.TargetHealthy(to.Int64p(2)))
	assert.Equal(t, 2, asg.TargetHealthy(to.Int64p(4)))
	assert.Equal(t, 3, asg.TargetHealthy(to.Int64p(6)))
	assert.Equal(t, 4, asg.TargetHealthy(to.Int64p(8)))
	assert.Equal(t, 5, asg.TargetHealthy(to.Int64p(10)))
}
