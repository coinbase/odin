package models

import (
	"github.com/coinbase/odin/aws"
	"github.com/coinbase/step/utils/to"
)

// Strategy describes the way in which Odin brings up instances in an Autoscaling Group
// pulling it out into this struct helps isolate code from the rest of the service
type Strategy struct {
	autoscaling             *AutoScalingConfig // This is populated on SetDefaults
	previousDesiredCapacity *int64             // This can be nil
}

////
// Health Report Methods
////

// TargetCapacity is the number of launched instances including the spread
func (strategy *Strategy) TargetCapacity() int64 {
	maxSize := strategy.maxSizeInt()
	dc := strategy.DesiredCapacity()
	spread := strategy.spreadFloat()

	tc := percent(dc, (1 + spread))
	return min(maxSize, tc)
}

// TargetHealthy is the number of instances the service needs to be Healthy
func (strategy *Strategy) TargetHealthy() int64 {
	minSize := strategy.minSizeInt()
	dc := strategy.DesiredCapacity()
	spread := strategy.spreadFloat()
	th := percent(dc, (1 - spread))

	return max(minSize, th)
}

// DesiredCapacity is the REAL amount of instances we want.
// This is altered for practicality by spread
func (strategy *Strategy) DesiredCapacity() int64 {
	minSize := strategy.minSizeInt()
	maxSize := strategy.maxSizeInt()
	previousDesiredCapacity := strategy.previousDesiredCapacity
	pc := int64(-1)
	if previousDesiredCapacity != nil {
		pc = *previousDesiredCapacity
	}

	// Scale down desired capacity if new max is lower than the previous dc
	desiredCapacity := min(pc, maxSize)
	// Scale up desired capacity, if the new min is higher than the previous dc
	return max(desiredCapacity, minSize)
}

////
// Init Methods
////

func (strategy *Strategy) InitialMinSize() *int64 {
	switch *strategy.autoscaling.Strategy {
	case "OneThenAllWithCanary":
		// "OneThenAllWithCanary" starts with 1
		return to.Int64p(1)
	case "25PercentStepRolloutNoCanary":
		// no instances yet
		return to.Int64p(fastRolloutRate(0, strategy.minSizeInt()))
	}

	// default case "AllAtOnce" is minSize
	return strategy.autoscaling.MinSize
}

func (strategy *Strategy) InitialDesiredCapacity() *int64 {
	switch *strategy.autoscaling.Strategy {
	case "OneThenAllWithCanary":
		// "OneThenAllWithCanary" starts with 1
		return to.Int64p(1)
	case "25PercentStepRolloutNoCanary":
		return to.Int64p(fastRolloutRate(0, strategy.TargetCapacity()))
	}

	// default case "AllAtOnce" is target capacity
	return to.Int64p(strategy.TargetCapacity())
}

////
// Flow Methods
////

func (strategy *Strategy) ReachedMaxTerminations(instances aws.Instances) bool {
	maxTermingInstances := strategy.maxTermsInt()

	switch *strategy.autoscaling.Strategy {
	case "OneThenAllWithCanary":
		// OneThenAllWithCanary during the canary it will exit if one terminates, otherwise default
		canarying := len(instances) <= 1
		if canarying {
			maxTermingInstances = 0
		}
	}

	// "AllAtOnce" "25PercentStepRolloutNoCanary" both error by default
	// If there are more terminating instances than allowed return true
	return int64(len(instances.TerminatingIDs())) > maxTermingInstances
}

func (strategy *Strategy) CalculateMinDesired(instances aws.Instances) (int64, int64) {
	switch *strategy.autoscaling.Strategy {
	case "OneThenAllWithCanary":
		// "OneThenAllWithCanary" if there is only one instance and it is healthy proceed
		canarying := len(instances) <= 1
		if !canarying {
			break
		} // Only continue if canarying

		canaryIsHealthy := len(instances.HealthyIDs()) == 1
		if canaryIsHealthy {
			break
		} // return default amounts if the canary is Healthy

		return 1, 1
	case "25PercentStepRolloutNoCanary":
		// 25PercentStepRolloutNoCanary will continually add 1/4 additional instances to those that are launching
		// until InitialMinSize and InitialDesiredCapacity
		return fastRolloutRate(len(instances), strategy.minSizeInt()), fastRolloutRate(len(instances), strategy.TargetCapacity())
	}

	// default case "AllAtOnce" is init values
	return strategy.minSizeInt(), strategy.TargetCapacity()
}

////
// Private Methods
////

func (strategy *Strategy) spreadFloat() float64 {
	if strategy.autoscaling.Spread == nil {
		return 0.2
	}
	return *strategy.autoscaling.Spread
}

func (strategy *Strategy) minSizeInt() int64 {
	if strategy.autoscaling.MinSize == nil {
		return 1
	}
	return int64(*strategy.autoscaling.MinSize)
}

func (strategy *Strategy) maxSizeInt() int64 {
	if strategy.autoscaling.MaxSize == nil {
		return 1
	}
	return int64(*strategy.autoscaling.MaxSize)
}

func (strategy *Strategy) maxTermsInt() int64 {
	if strategy.autoscaling.MaxTerminations == nil {
		return 0
	}
	return int64(*strategy.autoscaling.MaxTerminations)
}

////
// MATH
////

func min(x int64, y int64) int64 {
	if x < y {
		return x
	}
	return y
}

func max(x int64, y int64) int64 {
	if x > y {
		return x
	}
	return y
}

func percent(x int64, percent float64) int64 {
	return (x * int64(percent*100)) / 100
}

// 25PercentStepRolloutNoCanary

func fastRolloutRate(instanceCount int, baseAmount int64) int64 {
	// 1. Always return greater than 1
	// 2. Always return less than baseAmount
	// 3. return the instanceCount + 1/4 the baseAmount
	return max(1, min((int64(instanceCount)+baseAmount/4), baseAmount))
}
