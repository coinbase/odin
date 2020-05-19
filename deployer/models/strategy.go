package models

import (
	"github.com/coinbase/odin/aws"
	"github.com/coinbase/step/utils/to"
)

var STRATEGIES = []string{
	"AllAtOnce",
	"OneThenAllWithCanary",
	"25PercentStepRolloutNoCanary",
	"10PercentStepRolloutNoCanary",
	"10AtATimeNoCanary",
}

type StrategyType string

const (
	AllAtOnce StrategyType = "AllAtOnce"
	Canary                 = "Canary"
	Percent                = "Percent"
	Increment              = "Increment"
)

func NewStrategy(autoscaling *AutoScalingConfig, previousDesiredCapacity *int64) *Strategy {
	// Defaults
	s := &Strategy{
		name:                    *autoscaling.Strategy,
		sType:                   AllAtOnce,
		minSize:                 int64(1),
		maxSize:                 int64(1),
		maxTerminations:         int64(0),
		spread:                  0.2,
		previousDesiredCapacity: previousDesiredCapacity,
	}

	// Get Info from autoscaling
	if autoscaling.Spread != nil {
		s.spread = *autoscaling.Spread
	}

	if autoscaling.MinSize != nil {
		s.minSize = *autoscaling.MinSize
	}

	if autoscaling.MaxSize != nil {
		s.maxSize = *autoscaling.MaxSize
	}

	if autoscaling.MaxTerminations != nil {
		s.maxTerminations = *autoscaling.MaxTerminations
	}

	// Define the Strategy properties
	switch s.name {
	case "OneThenAllWithCanary":
		s.sType = Canary
	case "25PercentStepRolloutNoCanary":
		s.sType = Percent
		// 25% means release is divided into 4 steps
		s.rollOutSteps = 4
	case "10PercentStepRolloutNoCanary":
		s.sType = Percent
		// 10% means release divided into 10 stages
		s.rollOutSteps = 10
	case "10AtATimeNoCanary":
		s.sType = Increment
		s.rollOutSteps = float64(s.TargetCapacity()) / float64(10)
	}

	return s
}

// Strategy describes the way in which Odin brings up instances in an Autoscaling Group
// pulling it out into this struct helps isolate code from the rest of the service
type Strategy struct {
	name                    string
	sType                   StrategyType
	minSize                 int64
	maxSize                 int64
	maxTerminations         int64
	spread                  float64
	previousDesiredCapacity *int64 // This can be nil

	// For Percent and Increment types
	// This is the number of steps used to rollout all instances
	rollOutSteps float64
}

////
// Health Report Methods
////

// TargetCapacity is the number of launched instances including the spread
func (strategy *Strategy) TargetCapacity() int64 {
	maxSize := strategy.maxSize
	dc := strategy.DesiredCapacity()
	spread := strategy.spread

	tc := percent(dc, (1 + spread))
	return min(maxSize, tc)
}

// TargetHealthy is the number of instances the service needs to be Healthy
func (strategy *Strategy) TargetHealthy() int64 {
	minSize := strategy.minSize
	dc := strategy.DesiredCapacity()
	spread := strategy.spread
	th := percent(dc, (1 - spread))

	return max(minSize, th)
}

// DesiredCapacity is the REAL amount of instances we want.
// This is later altered for practicality by spread
func (strategy *Strategy) DesiredCapacity() int64 {
	minSize := strategy.minSize
	maxSize := strategy.maxSize
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
	switch strategy.sType {
	case Canary:
		// "OneThenAllWithCanary" starts with 1
		return to.Int64p(1)
	case Percent, Increment:
		// no instances yet
		return to.Int64p(fastRolloutRate(0, strategy.minSize, strategy.rollOutSteps))
	}

	// default case "AllAtOnce" is minSize
	return &strategy.minSize
}

func (strategy *Strategy) InitialDesiredCapacity() *int64 {
	switch strategy.sType {
	case Canary:
		// "OneThenAllWithCanary" starts with 1
		return to.Int64p(1)
	case Percent, Increment:
		return to.Int64p(fastRolloutRate(0, strategy.TargetCapacity(), strategy.rollOutSteps))
	}

	// default case "AllAtOnce" is target capacity
	return to.Int64p(strategy.TargetCapacity())
}

////
// Flow Methods
////

func (strategy *Strategy) ReachedMaxTerminations(instances aws.Instances) bool {
	maxTermingInstances := strategy.maxTerminations

	switch strategy.sType {
	case Canary:
		// OneThenAllWithCanary during the canary it will exit if one terminates, otherwise default
		canarying := len(instances) <= 1
		if canarying {
			maxTermingInstances = 0
		}
	}

	// Non Canaries just use maxTerms by default
	// If there are more terminating instances than allowed return true
	return int64(len(instances.TerminatingIDs())) > maxTermingInstances
}

func (strategy *Strategy) CalculateMinDesired(instances aws.Instances) (int64, int64) {
	switch strategy.sType {
	case Canary:
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
	case Percent, Increment:
		// Percent will continually add 1/strategy.rollOutSteps additional instances to those that are launching
		// until InitialMinSize and InitialDesiredCapacity
		minSize := fastRolloutRate(len(instances), strategy.minSize, strategy.rollOutSteps)
		dc := fastRolloutRate(len(instances), strategy.TargetCapacity(), strategy.rollOutSteps)
		return minSize, dc
	}

	// default case "AllAtOnce" is init values
	return strategy.minSize, strategy.TargetCapacity()
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

// 25PercentStepRolloutNoCanary and 10PercentStepRolloutNoCanary

func fastRolloutRate(instanceCount int, baseAmount int64, denominator float64) int64 {
	// 1. Always return greater than 1
	// 2. Always return less than baseAmount
	// 3. return the instanceCount + 1/4 the baseAmount

	// find the additional amount, always return more than 1
	additionalInstances := max(1, int64(float64(baseAmount)/denominator))

	// core return value
	amount := int64(instanceCount) + additionalInstances

	// Always return greater than 1, and less than baseAmount
	return max(1, min(amount, baseAmount))
}
