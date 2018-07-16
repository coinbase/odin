package aws

import (
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/aws/aws-sdk-go/service/elbv2"
)

const terminating = "terminating"
const unhealthy = "unhealthy"
const healthy = "healthy"

// Instances Map of instance id to state
type Instances map[string]string

// AddTargetGroupInstance add a target group instances
func (all Instances) AddTargetGroupInstance(thd *elbv2.TargetHealthDescription) {
	state := unhealthy
	if *thd.TargetHealth.State == "healthy" {
		state = healthy
	}
	all[*thd.Target.Id] = state
}

// AddASGInstance add a ASG instances
func (all Instances) AddASGInstance(i *autoscaling.Instance) {
	if i == nil || i.LifecycleState == nil {
		return
	}

	state := unhealthy

	if i.HealthStatus != nil && *i.HealthStatus == "Healthy" && *i.LifecycleState == "InService" {
		state = healthy
	}

	if (*i.LifecycleState)[0:4] == "Term" {
		state = terminating
	}

	all[*i.InstanceId] = state
}

// AddELBInstance add a ELB instances
func (all Instances) AddELBInstance(is *elb.InstanceState) {
	state := unhealthy
	if *is.State == "InService" {
		state = healthy
	}
	all[*is.InstanceId] = state
}

// HealthyUnhealthyTerming returns the numbers of states
func (all Instances) HealthyUnhealthyTerming() (int, int, int) {
	healthyc := 0
	unhealthyc := 0
	termingc := 0

	for _, state := range all {
		switch state {
		case healthy:
			healthyc++
		case unhealthy:
			unhealthyc++
		case terminating:
			termingc++
		}
	}

	return healthyc, unhealthyc, termingc
}

// InstanceIDs list of instance IDs
func (all Instances) InstanceIDs() []string {
	ids := []string{}

	for id := range all {
		ids = append(ids, id)
	}

	return ids
}

// HealthyIDs list of instances terminating
func (all Instances) UnhealthyIDs() []string {
	ids := []string{}
	for id, state := range all {
		if state == unhealthy {
			ids = append(ids, id)
		}
	}
	return ids
}

// HealthyIDs list of instances terminating
func (all Instances) HealthyIDs() []string {
	ids := []string{}
	for id, state := range all {
		if state == healthy {
			ids = append(ids, id)
		}
	}
	return ids
}

// TerminatingIDs list of instances terminating
func (all Instances) TerminatingIDs() []string {
	ids := []string{}
	for id, state := range all {
		if state == terminating {
			ids = append(ids, id)
		}
	}
	return ids
}

// MergeInstances merge new set of instances returns new set
func (all Instances) MergeInstances(update Instances) Instances {
	ret := Instances{}
	for id, state := range all {
		ret[id] = stateCompare(state, update[id])
	}

	return ret
}

func stateCompare(s1 string, s2 string) string {
	// terminating > unhealthy > healthy
	if s1 == healthy && s2 == healthy {
		// Both Healthy Return Healthy
		return healthy
	}

	if s1 == terminating || s2 == terminating {
		// Either Terming Return term
		return terminating
	}
	// Otherwise Unhealthy
	return unhealthy
}
