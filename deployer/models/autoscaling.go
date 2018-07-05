package models

import (
	"fmt"

	"github.com/coinbase/step/utils/to"
)

// AutoScalingConfig struct
type AutoScalingConfig struct {
	MinSize                *int64    `json:"min_size,omitempty"`
	MaxSize                *int64    `json:"max_size,omitempty"`
	MaxTerminations        *int64    `json:"max_terms,omitempty"`
	DefaultCooldown        *int64    `json:"default_cooldown,omitempty"`
	HealthCheckGracePeriod *int64    `json:"health_check_grace_period,omitempty"`
	Spread                 *float64  `json:"spread,omitempty"`
	Policies               []*Policy `json:"policies,omitempty"`
}

// MinSizeInt returns min size
func (a *AutoScalingConfig) MinSizeInt() int {
	if a.MinSize == nil {
		return 1
	}
	return int(*a.MinSize)
}

// MaxSizeInt returns max size
func (a *AutoScalingConfig) MaxSizeInt() int {
	if a.MaxSize == nil {
		return 1
	}
	return int(*a.MaxSize)
}

// MaxTerminationsInt returns maximum instances allowed to terminate
func (a *AutoScalingConfig) MaxTerminationsInt() int {
	if a.MaxTerminations == nil {
		return 0
	}
	return int(*a.MaxTerminations)
}

// ValidateAttributes validates attributes
func (a *AutoScalingConfig) ValidateAttributes() error {
	if a.MinSize == nil {
		return fmt.Errorf("Autoscaling MinSize is nil")
	}

	if a.MaxSize == nil {
		return fmt.Errorf("Autoscaling MaxSize is nil")
	}

	if a.Spread == nil {
		return fmt.Errorf("Autoscaling Spread is nil")
	}

	if *a.MinSize > *a.MaxSize {
		return fmt.Errorf("Autoscaling MinSize is Greater than MaxSize")
	}

	if *a.Spread < 0 || *a.Spread > 1 {
		return fmt.Errorf("Spread must be between 0 and 1")
	}

	for _, p := range a.Policies {
		if p == nil {
			return fmt.Errorf("Policy nil")
		}

		if err := p.ValidateAttributes(); err != nil {
			return err
		}
	}
	return nil
}

// SetDefaults assigns values
func (a *AutoScalingConfig) SetDefaults(serviceID *string) error {
	if a.MinSize == nil {
		a.MinSize = to.Int64p(1)
	}

	if a.MaxSize == nil {
		a.MaxSize = to.Int64p(1)
	}

	if a.Spread == nil {
		a.Spread = to.Float64p(0)
	}

	if a.MaxTerminations == nil {
		a.MaxTerminations = to.Int64p(0)
	}

	for _, p := range a.Policies {
		if p != nil {
			p.SetDefaults(serviceID)
		}
	}

	return nil
}

// DesiredCapacity returns default capacity
func (a *AutoScalingConfig) DesiredCapacity(previousDesiredCapacity *int64) int {
	return desiredCapacity(a.MinSizeInt(), a.MaxSizeInt(), pc(previousDesiredCapacity))
}

// TargetCapacity returns target capacity
func (a *AutoScalingConfig) TargetCapacity(previousDesiredCapacity *int64) int {
	return targetCapacity(a.MaxSizeInt(), a.DesiredCapacity(previousDesiredCapacity), *a.Spread)
}

// TargetHealthy returns target healthy
func (a *AutoScalingConfig) TargetHealthy(previousDesiredCapacity *int64) int {
	return targetHealthy(a.MinSizeInt(), a.DesiredCapacity(previousDesiredCapacity), *a.Spread)
}

// MATH

func desiredCapacity(minSize int, maxSize int, pc int) int {
	// Scale down desired capacity if new max is lower than the previous dc
	desiredCapacity := min(pc, maxSize)
	// Scale up desired capacity, if the new min is higher than the previous dc
	return max(desiredCapacity, minSize)
}

func targetHealthy(minSize int, dc int, spread float64) int {
	th := percent(dc, (1 - spread))
	return max(minSize, th)
}

func targetCapacity(maxSize int, dc int, spread float64) int {
	tc := percent(dc, (1 + spread))
	return min(maxSize, tc)
}

func pc(previousDesiredCapacity *int64) int {
	if previousDesiredCapacity == nil {
		return -1 // lower than min
	}
	return int(*previousDesiredCapacity)
}

func min(x int, y int) int {
	if x < y {
		return x
	}
	return y
}

func max(x int, y int) int {
	if x > y {
		return x
	}
	return y
}

func percent(x int, percent float64) int {
	return (x * int(percent*100)) / 100
}
