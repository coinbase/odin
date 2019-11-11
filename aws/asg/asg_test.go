package asg

import (
	"testing"

	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/coinbase/odin/aws/mocks"
	"github.com/coinbase/step/utils/to"
	"github.com/stretchr/testify/assert"
)

func Test_GetInstances(t *testing.T) {
	//func GetInstances(asgc aws.ASGAPI, asg_name *string) (aws.Instances, error) {
	asgc := &mocks.ASGClient{}
	_, err := GetInstances(asgc, to.Strp("asd"))
	assert.Error(t, err) // Not Found

	name := asgc.AddPreviousRuntimeResources("project", "config", "service", "release")
	ins, err := GetInstances(asgc, to.Strp(name))
	assert.NoError(t, err)
	assert.Equal(t, 1, len(ins))
}

func Test_ForProjectConfigNotReleaseIDServiceMap(t *testing.T) {
	// func ForProjectConfigNotReleaseIDServiceMap(asgc aws.ASGAPI, project_name *string, config_name *string, release_uuid *string) (map[string]*ASG, error) {
	asgc := &mocks.ASGClient{}
	services, err := ForProjectConfigNotReleaseIDServiceMap(asgc, to.Strp("project"), to.Strp("config"), to.Strp("release"))
	assert.NoError(t, err)
	assert.Equal(t, 0, len(services))

	asgc.AddPreviousRuntimeResources("project", "config", "service1", "release")
	asgc.AddPreviousRuntimeResources("project", "config", "service2", "release")
	asgc.AddPreviousRuntimeResources("project", "config", "service3", "not_release")
	asgc.AddPreviousRuntimeResources("not_project", "config", "service4", "release")
	asgc.AddPreviousRuntimeResources("project", "not_config", "service5", "release")

	services, err = ForProjectConfigNotReleaseIDServiceMap(asgc, to.Strp("project"), to.Strp("config"), to.Strp("release"))
	assert.NoError(t, err)
	assert.Equal(t, 1, len(services))
	assert.Equal(t, "service3", *services["service3"].ServiceName())
}

func Test_ForProjectConfigNotReleaseIDServiceMap_Error(t *testing.T) {
	// func ForProjectConfigNotReleaseIDServiceMap(asgc aws.ASGAPI, project_name *string, config_name *string, release_uuid *string) (map[string]*ASG, error) {
	asgc := &mocks.ASGClient{}
	asgc.AddPreviousRuntimeResources("project", "config", "service1", "release")
	asgc.AddPreviousRuntimeResources("project", "config", "service1", "release")

	_, err := ForProjectConfigNotReleaseIDServiceMap(asgc, to.Strp("project"), to.Strp("config"), to.Strp("not_release"))
	assert.Error(t, err)

}

func Test_ForProjectConfigNOTReleaseID(t *testing.T) {
	// func ForProjectConfigNOTReleaseID(asgc aws.ASGAPI, project_name *string, config_name *string, release_uuid *string) ([]*ASG, error) {
	asgc := &mocks.ASGClient{}
	asgs, err := ForProjectConfigNOTReleaseID(asgc, to.Strp("project"), to.Strp("config"), to.Strp("release"))
	assert.NoError(t, err)
	assert.Equal(t, 0, len(asgs))

	asgc.AddPreviousRuntimeResources("project", "config", "service1", "release")
	asgc.AddPreviousRuntimeResources("project", "config", "service2", "release")
	asgc.AddPreviousRuntimeResources("project", "config", "service3", "not_release")
	asgc.AddPreviousRuntimeResources("not_project", "config", "service4", "release")
	asgc.AddPreviousRuntimeResources("project", "not_config", "service5", "release")

	asgs, err = ForProjectConfigNOTReleaseID(asgc, to.Strp("project"), to.Strp("config"), to.Strp("release"))
	assert.NoError(t, err)
	assert.Equal(t, 1, len(asgs))
}

func Test_ForProjectConfigReleaseID(t *testing.T) {
	// func ForProjectConfigReleaseID(asgc aws.ASGAPI, project_name *string, config_name *string, release_uuid *string) ([]*ASG, error) {
	asgc := &mocks.ASGClient{}
	asgs, err := ForProjectConfigReleaseID(asgc, to.Strp("project"), to.Strp("config"), to.Strp("release"))
	assert.NoError(t, err)
	assert.Equal(t, 0, len(asgs))

	asgc.AddPreviousRuntimeResources("project", "config", "service1", "release")
	asgc.AddPreviousRuntimeResources("project", "config", "service2", "release")
	asgc.AddPreviousRuntimeResources("project", "config", "service3", "not_release")
	asgc.AddPreviousRuntimeResources("not_project", "config", "service4", "release")
	asgc.AddPreviousRuntimeResources("project", "not_config", "service5", "release")

	asgs, err = ForProjectConfigReleaseID(asgc, to.Strp("project"), to.Strp("config"), to.Strp("release"))
	assert.NoError(t, err)
	assert.Equal(t, 2, len(asgs))
}

func Test_Teardown(t *testing.T) {
	// func (s *ASG) Teardown(asgc aws.ASGAPI, cwc aws.CWAPI) error {
	asgc := &mocks.ASGClient{}
	cwc := &mocks.CWClient{}

	asgc.AddPreviousRuntimeResources("project", "config", "service1", "not_release")
	asgs, err := ForProjectConfigNOTReleaseID(asgc, to.Strp("project"), to.Strp("config"), to.Strp("release"))
	assert.NoError(t, err)
	assert.Equal(t, 1, len(asgs))

	err = asgs[0].Teardown(asgc, cwc)
	assert.NoError(t, err)
}

func Test_AttachedLBs(t *testing.T) {
	asgc := &mocks.ASGClient{}

	asgc.DescribeLoadBalancerTargetGroupsOutput = &autoscaling.DescribeLoadBalancerTargetGroupsOutput{
		LoadBalancerTargetGroups: []*autoscaling.LoadBalancerTargetGroupState{
			&autoscaling.LoadBalancerTargetGroupState{
				LoadBalancerTargetGroupARN: to.Strp("arn"),
				State: to.Strp("aaa"),
			},
		},
	}

	asgc.DescribeLoadBalancersOutput = &autoscaling.DescribeLoadBalancersOutput{
		LoadBalancers: []*autoscaling.LoadBalancerState{
			&autoscaling.LoadBalancerState{
				LoadBalancerName: to.Strp("arn"),
				State:            to.Strp("aaa"),
			},
		},
	}

	asgc.AddPreviousRuntimeResources("project", "config", "service1", "not_release")
	asgs, err := ForProjectConfigNOTReleaseID(asgc, to.Strp("project"), to.Strp("config"), to.Strp("release"))

	attached, err := asgs[0].AttachedLBs(asgc)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(attached))

	asgc.DescribeLoadBalancersOutput = &autoscaling.DescribeLoadBalancersOutput{
		LoadBalancers: []*autoscaling.LoadBalancerState{
			&autoscaling.LoadBalancerState{
				LoadBalancerName: to.Strp("arn"),
				State:            to.Strp("Removed"),
			},
		},
	}

	asgc.DescribeLoadBalancerTargetGroupsOutput = &autoscaling.DescribeLoadBalancerTargetGroupsOutput{
		LoadBalancerTargetGroups: []*autoscaling.LoadBalancerTargetGroupState{
			&autoscaling.LoadBalancerTargetGroupState{
				LoadBalancerTargetGroupARN: to.Strp("arn"),
				State: to.Strp("Removed"),
			},
		},
	}

	attached, err = asgs[0].AttachedLBs(asgc)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(attached))
}
