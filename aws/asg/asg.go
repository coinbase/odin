package asg

import (
	"fmt"

	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/coinbase/odin/aws"
	"github.com/coinbase/odin/aws/lc"
	"github.com/coinbase/step/utils/is"
	"github.com/coinbase/step/utils/to"
)

// ASG struct
type ASG struct {
	ProjectNameTag *string
	ConfigNameTag  *string
	ServiceNameTag *string
	ReleaseIDTag   *string
	ReleaseIdTag   *string

	MinSize         *int64
	DesiredCapacity *int64

	AutoScalingGroupName    *string
	LaunchConfigurationName *string

	LoadBalancerNames []*string
	TargetGroupARNs   []*string

	instances []*autoscaling.Instance
}

// ProjectName returns tag
func (s *ASG) ProjectName() *string {
	return s.ProjectNameTag
}

// ConfigName returns tag
func (s *ASG) ConfigName() *string {
	return s.ConfigNameTag
}

// ServiceName returns tag
func (s *ASG) ServiceName() *string {
	return s.ServiceNameTag
}

// AllowedService returns which service is allowed to attach to it
func (s *ASG) AllowedService() *string {
	return to.Strp(fmt.Sprintf("%s::%s::%s", *s.ProjectName(), *s.ConfigName(), *s.ServiceName()))
}

// ReleaseID returns tag
func (s *ASG) ReleaseID() *string {
	if !is.EmptyStr(s.ReleaseIDTag) {
		return s.ReleaseIDTag
	}
	return s.ReleaseIdTag
}

// ServiceID returns tag
func (s *ASG) ServiceID() *string {
	// Name of the AutoScalingGroup is the ServiceID
	return s.AutoScalingGroupName
}

//////
// Init
//////

func newASG(group *autoscaling.Group) *ASG {
	return &ASG{
		ProjectNameTag: aws.FetchASGTag(group.Tags, to.Strp("ProjectName")),
		ConfigNameTag:  aws.FetchASGTag(group.Tags, to.Strp("ConfigName")),
		ServiceNameTag: aws.FetchASGTag(group.Tags, to.Strp("ServiceName")),
		ReleaseIDTag:   aws.FetchASGTag(group.Tags, to.Strp("ReleaseID")),
		ReleaseIdTag:   aws.FetchASGTag(group.Tags, to.Strp("ReleaseId")),

		AutoScalingGroupName:    group.AutoScalingGroupName,
		LaunchConfigurationName: group.LaunchConfigurationName,

		LoadBalancerNames: group.LoadBalancerNames,
		TargetGroupARNs:   group.TargetGroupARNs,

		DesiredCapacity: group.DesiredCapacity,
		MinSize:         group.MinSize,

		instances: group.Instances,
	}
}

//////
// Healthy
//////

// GetInstances returns all instances on an ASG
func GetInstances(asgc aws.ASGAPI, asgName *string) (aws.Instances, *ASG, error) {
	group, err := findByName(asgc, asgName)
	if err != nil {
		return nil, nil, err
	}

	instances := aws.Instances{}

	for _, i := range group.instances {
		instances.AddASGInstance(i)
	}

	return instances, group, nil
}

func findByName(asgc aws.ASGAPI, asgName *string) (*ASG, error) {
	if asgName == nil {
		return nil, fmt.Errorf("Autoscaling group not found beause nil name")
	}

	asgs, err := findInAws(asgc,
		&autoscaling.DescribeAutoScalingGroupsInput{
			AutoScalingGroupNames: []*string{asgName},
		},
	)

	if err != nil {
		return nil, err
	}

	switch len(asgs) {
	case 0:
		return nil, fmt.Errorf("Autoscaling group %v not found", *asgName)
	case 1:
		return asgs[0], nil
	default:
		// Should never get here
		return nil, fmt.Errorf("Too many Autoscalings  found for %v", *asgName)
	}
}

//////////
// Find
//////////

// ForProjectConfigNotReleaseIDServiceMap finds all previous ASGs and returns them as a service map
// Will error if there is an ASG without a service name || two ASGs for a service
func ForProjectConfigNotReleaseIDServiceMap(asgc aws.ASGAPI, projectName *string, configName *string, releaseID *string) (map[string]*ASG, error) {
	asgs, err := ForProjectConfigNOTReleaseID(asgc, projectName, configName, releaseID)
	if err != nil {
		return nil, err
	}

	prevASGs := map[string]*ASG{}
	for _, asg := range asgs {
		sn := asg.ServiceName()
		if sn == nil {
			return nil, fmt.Errorf("Autoscaling Group found for Project with No Service Name %v", to.Strs(asg.ServiceID()))
		}

		if _, ok := prevASGs[*sn]; ok {
			return nil, fmt.Errorf("Found multiple ASGs for service %v -- %v", *sn, to.Strs(asg.ServiceID()))
		}

		prevASGs[*sn] = asg
	}

	return prevASGs, nil

}

// ForProjectConfigNOTReleaseID returns all ASGs not with the release ID
func ForProjectConfigNOTReleaseID(asgc aws.ASGAPI, projectName *string, configName *string, releaseID *string) ([]*ASG, error) {
	all, err := forProjectConfig(asgc, projectName, configName)
	if err != nil {
		return nil, err
	}

	asgs := []*ASG{}
	for _, asg := range all {
		if !aws.HasReleaseID(asg, releaseID) {
			asgs = append(asgs, asg)
		}
	}

	return asgs, nil
}

// ForProjectConfigReleaseID returns all ASGs with a release ID
func ForProjectConfigReleaseID(asgc aws.ASGAPI, projectName *string, configName *string, releaseID *string) ([]*ASG, error) {
	all, err := forProjectConfig(asgc, projectName, configName)
	if err != nil {
		return nil, err
	}

	asgs := []*ASG{}
	for _, asg := range all {
		if aws.HasReleaseID(asg, releaseID) {
			asgs = append(asgs, asg)
		}
	}

	return asgs, nil
}

func forProjectConfig(asgc aws.ASGAPI, projectName *string, configName *string) ([]*ASG, error) {
	all, err := findInAws(asgc, &autoscaling.DescribeAutoScalingGroupsInput{})
	if err != nil {
		return nil, err
	}

	asgs := []*ASG{}
	for _, asg := range all {
		if aws.HasProjectName(asg, projectName) && aws.HasConfigName(asg, configName) {
			asgs = append(asgs, asg)
		}
	}

	return asgs, nil
}

func findInAws(asgc aws.ASGAPI, params *autoscaling.DescribeAutoScalingGroupsInput) ([]*ASG, error) {
	allGroups := []*ASG{}

	params.SetMaxRecords(100)

	pagefn := func(page *autoscaling.DescribeAutoScalingGroupsOutput, lastPage bool) bool {
		for _, asg := range page.AutoScalingGroups {
			// Status either doesnt exist or is "Delete in progress", so filter here
			if asg.Status != nil {
				continue
			}

			allGroups = append(allGroups, newASG(asg))
		}
		// Return false only if last page
		return !lastPage
	}

	err := asgc.DescribeAutoScalingGroupsPages(params, pagefn)
	if err != nil {
		return nil, err
	}

	return allGroups, nil
}

//////////
// Destruction
//////////

func (s *ASG) Detach(asgc aws.ASGAPI) error {
	if len(s.LoadBalancerNames) > 0 {
		_, err := asgc.DetachLoadBalancers(&autoscaling.DetachLoadBalancersInput{
			AutoScalingGroupName: s.ServiceID(),
			LoadBalancerNames:    s.LoadBalancerNames,
		})

		if err != nil {
			return err
		}
	}

	if len(s.TargetGroupARNs) > 0 {
		_, err := asgc.DetachLoadBalancerTargetGroups(&autoscaling.DetachLoadBalancerTargetGroupsInput{
			AutoScalingGroupName: s.ServiceID(),
			TargetGroupARNs:      s.TargetGroupARNs,
		})

		if err != nil {
			return err
		}
	}

	return nil
}

func (s *ASG) AttachedLBs(asgc aws.ASGAPI) ([]string, error) {
	lbs := []string{}

	tgs, err := s.attachedTargetGroups(asgc)
	if err != nil {
		return lbs, err
	}

	clbs, err := s.attachedClassicLBs(asgc)
	if err != nil {
		return lbs, err
	}

	lbs = append(lbs, tgs...)
	lbs = append(lbs, clbs...)

	return lbs, err
}

func (s *ASG) attachedTargetGroups(asgc aws.ASGAPI) ([]string, error) {
	lbs := []string{}

	states, err := asgc.DescribeLoadBalancerTargetGroups(&autoscaling.DescribeLoadBalancerTargetGroupsInput{
		AutoScalingGroupName: s.ServiceID(),
	})

	if err != nil {
		return lbs, err
	}

	for _, targetGroup := range states.LoadBalancerTargetGroups {
		if *targetGroup.State == "Removed" {
			continue
		}

		lbs = append(lbs, *targetGroup.LoadBalancerTargetGroupARN)
	}

	return lbs, nil
}

func (s *ASG) attachedClassicLBs(asgc aws.ASGAPI) ([]string, error) {
	lbs := []string{}

	states, err := asgc.DescribeLoadBalancers(&autoscaling.DescribeLoadBalancersInput{
		AutoScalingGroupName: s.ServiceID(),
	})

	if err != nil {
		return lbs, err
	}

	for _, lb := range states.LoadBalancers {
		if *lb.State == "Removed" {
			continue
		}

		lbs = append(lbs, *lb.LoadBalancerName)
	}

	return lbs, nil
}

// Teardown deletes the ASG with launch config and alarms
func (s *ASG) Teardown(asgc aws.ASGAPI, cwc aws.CWAPI) error {
	// Delete Alarms
	alarms, err := s.alarmNames(asgc)
	if err != nil {
		return err
	}

	if err := s.teardownAlarms(cwc, alarms); err != nil {
		return err
	}

	// Delete Group
	if err := s.deleteGroup(asgc); err != nil {
		return err
	}

	// Delete Launch Config as well
	if err := lc.Teardown(asgc, s.LaunchConfigurationName); err != nil {
		return err
	}

	return nil
}

func (s *ASG) alarmNames(asgc aws.ASGAPI) ([]*string, error) {
	output, err := asgc.DescribePolicies(&autoscaling.DescribePoliciesInput{AutoScalingGroupName: s.AutoScalingGroupName})
	if err != nil {
		return nil, err
	}
	alarms := []*string{}
	for _, sp := range output.ScalingPolicies {
		for _, alarm := range sp.Alarms {
			alarms = append(alarms, alarm.AlarmName)
		}
	}

	return alarms, nil
}

func (s *ASG) teardownAlarms(cwc aws.CWAPI, alarms []*string) error {
	_, err := cwc.DeleteAlarms(&cloudwatch.DeleteAlarmsInput{AlarmNames: alarms})
	return err
}

func (s *ASG) deleteGroup(asgc aws.ASGAPI) error {
	_, err := asgc.DeleteAutoScalingGroup(&autoscaling.DeleteAutoScalingGroupInput{
		AutoScalingGroupName: s.ServiceID(),
		ForceDelete:          to.Boolp(true),
	})
	if err != nil {
		return err
	}
	return nil
}
