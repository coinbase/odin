package elb

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws/awserr"
	aws_elb "github.com/aws/aws-sdk-go/service/elb"
	"github.com/coinbase/odin/aws"
	"github.com/coinbase/step/utils/to"
)

// LoadBalancer struct
type LoadBalancer struct {
	ProjectNameTag   *string
	ConfigNameTag    *string
	ServiceNameTag   *string
	LoadBalancerName *string
}

// ProjectName returns tag
func (s *LoadBalancer) ProjectName() *string {
	return s.ProjectNameTag
}

// ConfigName returns tag
func (s *LoadBalancer) ConfigName() *string {
	return s.ConfigNameTag
}

// ServiceName returns tag
func (s *LoadBalancer) ServiceName() *string {
	return s.ServiceNameTag
}

// Name returns tag
func (s *LoadBalancer) Name() *string {
	return s.LoadBalancerName
}

// AllowedService returns tag
func (s *LoadBalancer) AllowedService() *string {
	return to.Strp(fmt.Sprintf("%s::%s::%s", *s.ProjectName(), *s.ConfigName(), *s.ServiceName()))
}

///////
// Healthy
///////

// GetInstances returns a list of specific instances on the ELB
func GetInstances(elbc aws.ELBAPI, name *string, instances []string) (aws.Instances, error) {
	instanceStates, err := instanceStates(elbc, name, instances)

	if err != nil {
		return nil, err
	}

	elbInstances := aws.Instances{}
	for _, is := range instanceStates {
		elbInstances.AddELBInstance(is)
	}

	return elbInstances, nil
}

func createDescribeInstanceHealthInput(name *string, instances []string) *aws_elb.DescribeInstanceHealthInput {
	awsInstances := []*aws_elb.Instance{}
	for _, id := range instances {
		awsInstances = append(awsInstances, &aws_elb.Instance{InstanceId: to.Strp(id)})
	}

	return &aws_elb.DescribeInstanceHealthInput{
		LoadBalancerName: name,
		Instances:        awsInstances,
	}
}

func instanceStates(elbc aws.ELBAPI, name *string, instances []string) ([]*aws_elb.InstanceState, error) {

	healthOutput, err := elbc.DescribeInstanceHealth(createDescribeInstanceHealthInput(name, instances))

	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case aws_elb.ErrCodeInvalidEndPointException:
				// Error occurs if instance is not yet (or no longer) attached to the ELB
				return []*aws_elb.InstanceState{}, nil
			}
		}
		return nil, err
	}

	return healthOutput.InstanceStates, nil
}

///////
// Find
///////

// FindAll returns ELBs with names
func FindAll(elbc aws.ELBAPI, names []*string) ([]*LoadBalancer, error) {
	elbs := []*LoadBalancer{}
	for _, name := range names {
		elb, err := find(elbc, name)
		if err != nil {
			return nil, err
		}
		elbs = append(elbs, elb)
	}

	return elbs, nil
}

func find(elbc aws.ELBAPI, name *string) (*LoadBalancer, error) {

	elbDesc, err := findAwsByName(elbc, name)

	if err != nil {
		return nil, err
	}

	tags, err := findAwsTagsByName(elbc, name)

	if err != nil {
		return nil, err
	}

	return &LoadBalancer{
		ProjectNameTag:   aws.FetchELBTag(tags, to.Strp("ProjectName")),
		ConfigNameTag:    aws.FetchELBTag(tags, to.Strp("ConfigName")),
		ServiceNameTag:   aws.FetchELBTag(tags, to.Strp("ServiceName")),
		LoadBalancerName: elbDesc.LoadBalancerName,
	}, nil
}

func findAwsByName(elbc aws.ELBAPI, name *string) (*aws_elb.LoadBalancerDescription, error) {
	elbsOutput, err := elbc.DescribeLoadBalancers(&aws_elb.DescribeLoadBalancersInput{
		LoadBalancerNames: []*string{name},
	})

	if err != nil {
		return nil, err
	}

	if len(elbsOutput.LoadBalancerDescriptions) != 1 {
		return nil, fmt.Errorf("LoadBalancer Not Found")
	}

	if *elbsOutput.LoadBalancerDescriptions[0].LoadBalancerName != *name {
		return nil, fmt.Errorf("LoadBalancer Not Found")
	}

	return elbsOutput.LoadBalancerDescriptions[0], nil
}

func findAwsTagsByName(elbc aws.ELBAPI, name *string) ([]*aws_elb.Tag, error) {
	tagsOutput, err := elbc.DescribeTags(&aws_elb.DescribeTagsInput{
		LoadBalancerNames: []*string{name},
	})

	if err != nil {
		return nil, err
	}

	if len(tagsOutput.TagDescriptions) != 1 {
		return nil, fmt.Errorf("LoadBalancer Not Found")
	}

	if *tagsOutput.TagDescriptions[0].LoadBalancerName != *name {
		return nil, fmt.Errorf("LoadBalancer Not Found")
	}

	return tagsOutput.TagDescriptions[0].Tags, nil

}
