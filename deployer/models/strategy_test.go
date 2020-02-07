package models

import (
	"fmt"
	"testing"

	"github.com/coinbase/odin/aws"
	"github.com/coinbase/step/utils/to"
	"github.com/stretchr/testify/assert"
)

func simpleStrategy(min, max int64, dc *int64, spread *float64) *Strategy {
	return &Strategy{
		autoscaling: &AutoScalingConfig{
			MinSize:         to.Int64p(min),
			MaxSize:         to.Int64p(max),
			Spread:          spread,
			MaxTerminations: to.Int64p(0),
			Strategy:        to.Strp("AllAtOnce"),
		},
		previousDesiredCapacity: dc,
	}
}

////
// Invariant Methods
////

func Test_Strategy_DesiredCapacity(t *testing.T) {
	assert.EqualValues(t, 1, simpleStrategy(1, 3, to.Int64p(-1), nil).DesiredCapacity())
	assert.EqualValues(t, 1, simpleStrategy(1, 3, to.Int64p(1), nil).DesiredCapacity())
	assert.EqualValues(t, 2, simpleStrategy(1, 3, to.Int64p(2), nil).DesiredCapacity())
	assert.EqualValues(t, 3, simpleStrategy(1, 3, to.Int64p(3), nil).DesiredCapacity())
}

func Test_Strategy_TargetCapacity(t *testing.T) {
	assert.EqualValues(t, 1, simpleStrategy(1, 1, to.Int64p(1), to.Float64p(1)).TargetCapacity())
	assert.EqualValues(t, 3, simpleStrategy(1, 3, to.Int64p(2), to.Float64p(1)).TargetCapacity())

	assert.EqualValues(t, 1, simpleStrategy(1, 10, nil, to.Float64p(0.5)).TargetCapacity())
	assert.EqualValues(t, 3, simpleStrategy(1, 10, to.Int64p(2), to.Float64p(0.5)).TargetCapacity())
	assert.EqualValues(t, 6, simpleStrategy(1, 10, to.Int64p(4), to.Float64p(0.5)).TargetCapacity())
	assert.EqualValues(t, 9, simpleStrategy(1, 10, to.Int64p(6), to.Float64p(0.5)).TargetCapacity())
	assert.EqualValues(t, 10, simpleStrategy(1, 10, to.Int64p(8), to.Float64p(0.5)).TargetCapacity())
	assert.EqualValues(t, 10, simpleStrategy(1, 10, to.Int64p(10), to.Float64p(0.5)).TargetCapacity())
}

func Test_Strategy_TargetHealthy(t *testing.T) {
	assert.EqualValues(t, 1, simpleStrategy(1, 1, to.Int64p(1), to.Float64p(0)).TargetHealthy())
	assert.EqualValues(t, 1, simpleStrategy(1, 10, to.Int64p(3), to.Float64p(1)).TargetHealthy())

	assert.EqualValues(t, 1, simpleStrategy(1, 10, nil, to.Float64p(0.5)).TargetHealthy())
	assert.EqualValues(t, 1, simpleStrategy(1, 10, to.Int64p(2), to.Float64p(0.5)).TargetHealthy())
	assert.EqualValues(t, 2, simpleStrategy(1, 10, to.Int64p(4), to.Float64p(0.5)).TargetHealthy())
	assert.EqualValues(t, 3, simpleStrategy(1, 10, to.Int64p(6), to.Float64p(0.5)).TargetHealthy())
	assert.EqualValues(t, 4, simpleStrategy(1, 10, to.Int64p(8), to.Float64p(0.5)).TargetHealthy())
	assert.EqualValues(t, 5, simpleStrategy(1, 10, to.Int64p(10), to.Float64p(0.5)).TargetHealthy())
}

////
// Strategy Methods
////

var oneGood = aws.Instances{"one": "healthy"}

var oneUnHealthy = aws.Instances{"one": "unhealthy"}

var oneTerming = aws.Instances{"one": "terminating"}
var twoTerming = aws.Instances{"one": "terminating", "two": "terminating"}
var oneOfTwoTerming = aws.Instances{"one": "terminating", "two": "healthy"}

var twoLaunching = aws.Instances{"one": "unhealthy", "two": "unhealthy"}

func complexSrategy(strat string) *Strategy {
	asg := &AutoScalingConfig{
		MinSize:         to.Int64p(1),
		MaxSize:         to.Int64p(50),
		MaxTerminations: to.Int64p(1),
		Spread:          to.Float64p(0), // Remove Spread from calculations
		Strategy:        to.Strp(strat),
	}

	asg.SetDefaults(to.Strp("service_id"), to.Intp(30))

	return &Strategy{autoscaling: asg, previousDesiredCapacity: to.Int64p(25)}
}

////
// AllAtOnce, i.e. the default strategy
////

func Test_Strategy_AllAtOnce_InitValues(t *testing.T) {
	// AllAtOnce does not change throughout a deploy
	// So initial values are the same as target values

	strat := complexSrategy("AllAtOnce")

	assert.EqualValues(t, *strat.InitialMinSize(), strat.minSizeInt())
	assert.EqualValues(t, *strat.InitialDesiredCapacity(), strat.TargetCapacity())
}

func Test_Strategy_AllAtOnce_Termination(t *testing.T) {
	// AllAtOnce does not change throughout the deploy
	// ReachedMaxTerminations
	strat := complexSrategy("AllAtOnce")

	// unless there are two terminating then we didnt reach the limit
	assert.EqualValues(t, false, strat.ReachedMaxTerminations(oneGood))
	assert.EqualValues(t, false, strat.ReachedMaxTerminations(oneTerming))
	assert.EqualValues(t, false, strat.ReachedMaxTerminations(oneOfTwoTerming))
	assert.EqualValues(t, true, strat.ReachedMaxTerminations(twoTerming))
}

var allAtOnceCalcs = []struct {
	instances aws.Instances
	min       int64
	dc        int64
}{
	{
		instances: oneGood,
		min:       1,
		dc:        25,
	},
	{
		instances: oneUnHealthy,
		min:       1,
		dc:        25,
	},
	{
		instances: twoLaunching,
		min:       1,
		dc:        25,
	},
}

func Test_Strategy_AllAtOnce_Min_And_Desired(t *testing.T) {
	for i, test := range allAtOnceCalcs {
		t.Run(fmt.Sprintf("test: %v", i), func(t *testing.T) {
			strat := complexSrategy("AllAtOnce")

			min, dc := strat.CalculateMinDesired(test.instances)

			assert.EqualValues(t, test.min, min)
			assert.EqualValues(t, test.dc, dc)
		})
	}
}

////
// OneThenAllWithCanary, i.e. canary
////

func Test_Strategy_OneThenAllWithCanary_InitValues(t *testing.T) {
	// OneThenAllWithCanary does not change throughout a deploy
	// so intial and dc are 1
	strat := complexSrategy("OneThenAllWithCanary")

	assert.EqualValues(t, *strat.InitialMinSize(), 1)
	assert.EqualValues(t, *strat.InitialDesiredCapacity(), 1)
}

func Test_Strategy_OneThenAllWithCanary_Termination(t *testing.T) {
	// OneThenAllWithCanary has a max term count of 1 when there is only 1 instance
	// ReachedMaxTerminations
	strat := complexSrategy("OneThenAllWithCanary")

	assert.EqualValues(t, false, strat.ReachedMaxTerminations(oneGood))
	// If there is only 1 instance and it is terminating then we think that the canary it terming
	assert.EqualValues(t, true, strat.ReachedMaxTerminations(oneTerming))
	assert.EqualValues(t, false, strat.ReachedMaxTerminations(oneOfTwoTerming))
	assert.EqualValues(t, true, strat.ReachedMaxTerminations(twoTerming))
}

var oneAtATimeCalcs = []struct {
	instances aws.Instances
	min       int64
	dc        int64
}{
	// If there is one unhealthy we are still canarying
	{
		instances: oneUnHealthy,
		min:       1,
		dc:        1,
	},
	// If there is one good then we can stop canarying
	{
		instances: oneGood,
		min:       1,
		dc:        25,
	},
	// If there is two launching continue launching
	{
		instances: twoLaunching,
		min:       1,
		dc:        25,
	},
}

func Test_Strategy_OneThenAllWithCanary_Min_And_Desired(t *testing.T) {
	for i, test := range oneAtATimeCalcs {
		t.Run(fmt.Sprintf("test: %v", i), func(t *testing.T) {
			strat := complexSrategy("OneThenAllWithCanary")

			min, dc := strat.CalculateMinDesired(test.instances)

			assert.EqualValues(t, test.min, min)
			assert.EqualValues(t, test.dc, dc)
		})
	}
}

////
// 25PercentStepRolloutNoCanary, i.e. launching in quarters
////

func Test_Strategy_25StepRolloutNoCanary_Rate(t *testing.T) {
	// Always return at least 1
	assert.EqualValues(t, 1, fastRolloutRate(0, 1))
	assert.EqualValues(t, 1, fastRolloutRate(0, 2))
	assert.EqualValues(t, 1, fastRolloutRate(0, 4))

	// Dont get stuck on low numbers
	assert.EqualValues(t, 1, fastRolloutRate(1, 1))
	assert.EqualValues(t, 2, fastRolloutRate(1, 2))

	// Never return greater than the baseAmount
	assert.EqualValues(t, 5, fastRolloutRate(100, 5))
	assert.EqualValues(t, 10, fastRolloutRate(10, 10))

	// return a quarter + the instance amount
	assert.EqualValues(t, 2, fastRolloutRate(1, 4))
	assert.EqualValues(t, 3, fastRolloutRate(2, 4))
}

func Test_Strategy_25StepRolloutNoCanary_InitValues(t *testing.T) {
	// 25PercentStepRolloutNoCanary does not change throughout a deploy
	// So initial values are the same as target values

	strat := complexSrategy("25PercentStepRolloutNoCanary")

	assert.EqualValues(t, *strat.InitialMinSize(), 1)
	assert.EqualValues(t, *strat.InitialDesiredCapacity(), 6) // 25/4
}

func Test_Strategy_25StepRolloutNoCanary_Termination(t *testing.T) {
	// 25PercentStepRolloutNoCanary does not change throughout the deploy
	// ReachedMaxTerminations
	strat := complexSrategy("25PercentStepRolloutNoCanary")

	// unless there are two terminating then we didnt reach the limit
	assert.EqualValues(t, false, strat.ReachedMaxTerminations(oneGood))
	assert.EqualValues(t, false, strat.ReachedMaxTerminations(oneTerming))
	assert.EqualValues(t, false, strat.ReachedMaxTerminations(oneOfTwoTerming))
	assert.EqualValues(t, true, strat.ReachedMaxTerminations(twoTerming))
}

var fastRolloutCalcs = []struct {
	instances aws.Instances
	min       int64
	dc        int64
}{
	{
		instances: oneGood,
		min:       1,
		dc:        7, // 25/4 + 1
	},
	{
		instances: oneUnHealthy,
		min:       1,
		dc:        7, // 25/4 + 1
	},
	{
		instances: twoLaunching,
		min:       1,
		dc:        8, // 25/4 + 2
	},
}

func Test_Strategy_25StepRolloutNoCanary_Min_And_Desired(t *testing.T) {
	for i, test := range fastRolloutCalcs {
		t.Run(fmt.Sprintf("test: %v", i), func(t *testing.T) {
			strat := complexSrategy("25PercentStepRolloutNoCanary")

			min, dc := strat.CalculateMinDesired(test.instances)

			assert.EqualValues(t, test.min, min)
			assert.EqualValues(t, test.dc, dc)
		})
	}
}
