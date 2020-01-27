package models

import (
	"fmt"

	"github.com/coinbase/step/utils/is"
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

	// Type one of "AllAtOnce" "OneThenAllWithCanary" "25PercentStepRolloutNoCanary"
	Strategy *string `json:"strategy,omitempty"`
}

// ValidateAttributes validates attributes
func (a *AutoScalingConfig) ValidateAttributes() error {
	if a.Strategy == nil {
		return fmt.Errorf("Autoscaling Strategy nil")
	}

	switch *a.Strategy {
	case "AllAtOnce", "OneThenAllWithCanary", "25PercentStepRolloutNoCanary":
		//skip
	default:
		return fmt.Errorf("Autoscaling Strategy must be either 'AllAtOnce', 'OneThenAllWithCanary', '25PercentStepRolloutNoCanary'")
	}

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

	policyNames := []*string{}

	for _, p := range a.Policies {
		if p == nil {
			return fmt.Errorf("Policy nil")
		}

		if err := p.ValidateAttributes(); err != nil {
			return err
		}

		policyNames = append(policyNames, p.Name())
	}

	if !is.UniqueStrp(policyNames) {
		return fmt.Errorf("Policy Names not Unique")
	}

	return nil
}

// SetDefaults assigns values
func (a *AutoScalingConfig) SetDefaults(serviceID *string, timeout *int) error {

	if a.Strategy == nil {
		a.Strategy = to.Strp("AllAtOnce")
	}

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

	if a.HealthCheckGracePeriod == nil && timeout != nil {
		// Increase the HealthCheckGracePeriod from default to timeout if not specified
		// This ensures instaces are not terminated early while we are waiting for healthy status
		// Downside: instances might not be terminated after the deploy finished due to bad health
		a.HealthCheckGracePeriod = to.Int64p(int64(*timeout))
	} else if a.HealthCheckGracePeriod != nil && timeout != nil {
		// There is no reason for HealthCheckGracePeriod to be above timeout
		// It could cause a successful deploy to not term unhealthy instances after deployer
		// For unsuccessful deploys it makes no difference
		a.HealthCheckGracePeriod = to.Int64p(
			min(
				*a.HealthCheckGracePeriod,
				int64(*timeout),
			))
	}

	for _, p := range a.Policies {
		if p != nil {
			p.SetDefaults(serviceID)
		}
	}

	return nil
}
