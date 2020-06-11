package mocks

import (
	"fmt"

	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/coinbase/odin/aws"
	"github.com/coinbase/step/utils/to"
)

// DescribeAutoScalingGroupResponse returns
type DescribeAutoScalingGroupResponse struct {
	Resp  *autoscaling.DescribeAutoScalingGroupsOutput
	Error error
}

// DescribeLaunchConfigurationsResponse returns
type DescribeLaunchConfigurationsResponse struct {
	Resp  *autoscaling.DescribeLaunchConfigurationsOutput
	Error error
}

// DescribePoliciesResponse returns
type DescribePoliciesResponse struct {
	Resp  *autoscaling.DescribePoliciesOutput
	Error error
}

// ASGClient returns
type ASGClient struct {
	aws.ASGAPI
	DescribeAutoScalingGroupsPageResp []DescribeAutoScalingGroupResponse
	DescribeLaunchConfigurationsResp  map[string]*DescribeLaunchConfigurationsResponse
	DescribePoliciesResp              map[string]*DescribePoliciesResponse

	DescribeLoadBalancerTargetGroupsOutput *autoscaling.DescribeLoadBalancerTargetGroupsOutput
	DescribeLoadBalancersOutput            *autoscaling.DescribeLoadBalancersOutput

	UpdateAutoScalingGroupLastInput *autoscaling.UpdateAutoScalingGroupInput
	DetachLoadBalancersError        error
}

func (m *ASGClient) init() {
	if m.DescribeAutoScalingGroupsPageResp == nil {
		m.DescribeAutoScalingGroupsPageResp = []DescribeAutoScalingGroupResponse{}
	}

	if m.DescribeLaunchConfigurationsResp == nil {
		m.DescribeLaunchConfigurationsResp = map[string]*DescribeLaunchConfigurationsResponse{}
	}

	if m.DescribePoliciesResp == nil {
		m.DescribePoliciesResp = map[string]*DescribePoliciesResponse{}
	}
}

// MakeMockASG returns
func MakeMockASG(name string, projetName string, configName string, serviceName string, releaseID string) *autoscaling.Group {
	return &autoscaling.Group{
		AutoScalingGroupName: to.Strp(name),
		Instances:            MakeMockASGInstances(1, 0, 0),
		LoadBalancerNames:    []*string{to.Strp("elb")},
		TargetGroupARNs:      []*string{to.Strp("tg")},

		MinSize:         to.Int64p(1),
		MaxSize:         to.Int64p(3),
		DesiredCapacity: to.Int64p(1),
		Tags: []*autoscaling.TagDescription{
			&autoscaling.TagDescription{Key: to.Strp("ProjectName"), Value: to.Strp(projetName)},
			&autoscaling.TagDescription{Key: to.Strp("ConfigName"), Value: to.Strp(configName)},
			&autoscaling.TagDescription{Key: to.Strp("ServiceName"), Value: to.Strp(serviceName)},
			&autoscaling.TagDescription{Key: to.Strp("ReleaseID"), Value: to.Strp(releaseID)},
		},
	}
}

// MakeMockASGInstances returns
func MakeMockASGInstances(healthy int, unhealthy int, terming int) []*autoscaling.Instance {
	ins := []*autoscaling.Instance{}
	x := 0
	for i := 0; i < healthy; i++ {
		x++
		ins = append(ins, &autoscaling.Instance{
			InstanceId:     to.Strp(fmt.Sprintf("InstanceId%v", x)),
			HealthStatus:   to.Strp("Healthy"),
			LifecycleState: to.Strp("InService"),
		})
	}

	for i := 0; i < unhealthy; i++ {
		x++
		ins = append(ins, &autoscaling.Instance{
			InstanceId:     to.Strp(fmt.Sprintf("InstanceId%v", x)),
			HealthStatus:   to.Strp("Unhealthy"),
			LifecycleState: to.Strp("Waiting"),
		})
	}

	for i := 0; i < terming; i++ {
		x++
		ins = append(ins, &autoscaling.Instance{
			InstanceId:     to.Strp(fmt.Sprintf("InstanceId%v", x)),
			HealthStatus:   to.Strp("Terminating"),
			LifecycleState: to.Strp("Terminating"),
		})
	}
	return ins
}

// AddASG returns
func (m *ASGClient) AddASG(asg *autoscaling.Group) {
	m.init()
	m.DescribeAutoScalingGroupsPageResp = append(m.DescribeAutoScalingGroupsPageResp,
		DescribeAutoScalingGroupResponse{
			Resp: &autoscaling.DescribeAutoScalingGroupsOutput{
				AutoScalingGroups: []*autoscaling.Group{
					asg,
				},
			},
		},
	)
}

// AddPreviousRuntimeResources returns
func (m *ASGClient) AddPreviousRuntimeResources(projectName string, configName string, serviceName string, releaseID string) string {
	m.init()

	name := fmt.Sprintf("%v-%v-%v-%v", projectName, configName, serviceName, releaseID)

	m.AddASG(MakeMockASG(name, projectName, configName, serviceName, releaseID))

	m.DescribeLaunchConfigurationsResp[name] = &DescribeLaunchConfigurationsResponse{
		Resp: &autoscaling.DescribeLaunchConfigurationsOutput{
			LaunchConfigurations: []*autoscaling.LaunchConfiguration{
				&autoscaling.LaunchConfiguration{LaunchConfigurationName: to.Strp(name)},
			},
		},
	}

	m.DescribePoliciesResp[name] = &DescribePoliciesResponse{
		Resp: &autoscaling.DescribePoliciesOutput{
			ScalingPolicies: []*autoscaling.ScalingPolicy{
				&autoscaling.ScalingPolicy{
					Alarms: []*autoscaling.Alarm{
						&autoscaling.Alarm{
							AlarmName: to.Strp("VeryEmbeddedAlarm"),
						},
					},
				},
			},
		},
	}

	return name
}

// DescribeAutoScalingGroupsPages returns
func (m *ASGClient) DescribeAutoScalingGroupsPages(input *autoscaling.DescribeAutoScalingGroupsInput, fn func(*autoscaling.DescribeAutoScalingGroupsOutput, bool) bool) error {
	m.init()
	// Loop through all autoscaling groups, 1 per page
	var cont bool
	for _, page := range m.DescribeAutoScalingGroupsPageResp {
		if page.Error != nil {
			return page.Error
		}

		cont = fn(page.Resp, false)

		if !cont {
			return fmt.Errorf("Should always end here")
		}
	}

	cont = fn(&autoscaling.DescribeAutoScalingGroupsOutput{}, true)

	if cont {
		return fmt.Errorf("Should always end here")
	}

	return nil
}

// DeleteAutoScalingGroup returns
func (m *ASGClient) DeleteAutoScalingGroup(input *autoscaling.DeleteAutoScalingGroupInput) (*autoscaling.DeleteAutoScalingGroupOutput, error) {
	return nil, nil
}

// CreateAutoScalingGroup returns
func (m *ASGClient) CreateAutoScalingGroup(input *autoscaling.CreateAutoScalingGroupInput) (*autoscaling.CreateAutoScalingGroupOutput, error) {
	return nil, nil
}

// DescribeLaunchConfigurations returns
func (m *ASGClient) DescribeLaunchConfigurations(in *autoscaling.DescribeLaunchConfigurationsInput) (*autoscaling.DescribeLaunchConfigurationsOutput, error) {
	m.init()
	lcName := in.LaunchConfigurationNames[0]
	resp := m.DescribeLaunchConfigurationsResp[*lcName]
	if resp == nil {
		return &autoscaling.DescribeLaunchConfigurationsOutput{LaunchConfigurations: []*autoscaling.LaunchConfiguration{}}, nil
	}
	return resp.Resp, resp.Error
}

// CreateLaunchConfiguration returns
func (m *ASGClient) CreateLaunchConfiguration(input *autoscaling.CreateLaunchConfigurationInput) (*autoscaling.CreateLaunchConfigurationOutput, error) {
	return nil, nil
}

// DeleteLaunchConfiguration returns
func (m *ASGClient) DeleteLaunchConfiguration(input *autoscaling.DeleteLaunchConfigurationInput) (*autoscaling.DeleteLaunchConfigurationOutput, error) {
	return nil, nil
}

// DescribePolicies returns
func (m *ASGClient) DescribePolicies(in *autoscaling.DescribePoliciesInput) (*autoscaling.DescribePoliciesOutput, error) {
	m.init()
	resp := m.DescribePoliciesResp[*in.AutoScalingGroupName]
	if resp == nil {
		return &autoscaling.DescribePoliciesOutput{}, nil
	}
	return resp.Resp, resp.Error
}

// EnableMetricsCollection returns
func (m *ASGClient) EnableMetricsCollection(input *autoscaling.EnableMetricsCollectionInput) (*autoscaling.EnableMetricsCollectionOutput, error) {
	return nil, nil
}

// PutScalingPolicy returns
func (m *ASGClient) PutScalingPolicy(input *autoscaling.PutScalingPolicyInput) (*autoscaling.PutScalingPolicyOutput, error) {
	return &autoscaling.PutScalingPolicyOutput{PolicyARN: to.Strp("arn")}, nil
}

func (m *ASGClient) DetachLoadBalancers(input *autoscaling.DetachLoadBalancersInput) (*autoscaling.DetachLoadBalancersOutput, error) {
	return nil, m.DetachLoadBalancersError
}

func (m *ASGClient) DetachLoadBalancerTargetGroups(input *autoscaling.DetachLoadBalancerTargetGroupsInput) (*autoscaling.DetachLoadBalancerTargetGroupsOutput, error) {
	return nil, nil
}

func (m *ASGClient) DescribeLoadBalancerTargetGroups(input *autoscaling.DescribeLoadBalancerTargetGroupsInput) (*autoscaling.DescribeLoadBalancerTargetGroupsOutput, error) {
	if m.DescribeLoadBalancerTargetGroupsOutput != nil {
		return m.DescribeLoadBalancerTargetGroupsOutput, nil
	}
	return &autoscaling.DescribeLoadBalancerTargetGroupsOutput{}, nil
}

func (m *ASGClient) DescribeLoadBalancers(input *autoscaling.DescribeLoadBalancersInput) (*autoscaling.DescribeLoadBalancersOutput, error) {
	if m.DescribeLoadBalancersOutput != nil {
		return m.DescribeLoadBalancersOutput, nil
	}
	return &autoscaling.DescribeLoadBalancersOutput{}, nil
}

func (m *ASGClient) UpdateAutoScalingGroup(input *autoscaling.UpdateAutoScalingGroupInput) (*autoscaling.UpdateAutoScalingGroupOutput, error) {
	m.UpdateAutoScalingGroupLastInput = input
	return nil, nil
}
