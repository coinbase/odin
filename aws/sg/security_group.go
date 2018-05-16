package sg

import (
	"fmt"

	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/coinbase/odin/aws"
	"github.com/coinbase/step/utils/to"
)

// SecurityGroup struct
type SecurityGroup struct {
	ProjectNameTag *string
	ConfigNameTag  *string
	ServiceNameTag *string
	GroupID        *string
}

// ProjectName returns tag
func (s *SecurityGroup) ProjectName() *string {
	return s.ProjectNameTag
}

// ConfigName returns tag
func (s *SecurityGroup) ConfigName() *string {
	return s.ConfigNameTag
}

// ServiceName returns tag
func (s *SecurityGroup) ServiceName() *string {
	return s.ServiceNameTag
}

// Find returns the security groups with tags
func Find(ec2Client aws.EC2API, nameTags []*string) ([]*SecurityGroup, error) {
	filters := []*ec2.Filter{
		&ec2.Filter{
			Name:   to.Strp("tag-key"),
			Values: []*string{to.Strp("Name")},
		},
		&ec2.Filter{
			Name:   to.Strp("tag-value"),
			Values: nameTags,
		},
	}

	output, err := ec2Client.DescribeSecurityGroups(&ec2.DescribeSecurityGroupsInput{
		Filters:    filters,
		MaxResults: to.Int64p(5), // Smallest allowed value returns
	})

	if err != nil {
		return nil, err
	}

	sgs := newSGs(output.SecurityGroups)
	switch len(sgs) {
	case len(nameTags):
		return sgs, nil
	default:
		return nil, fmt.Errorf("Number of Security Groups %v/%v", len(sgs), len(nameTags))
	}
}

func newSGs(output []*ec2.SecurityGroup) []*SecurityGroup {
	sgs := []*SecurityGroup{}
	for _, sg := range output {
		sgs = append(sgs, &SecurityGroup{
			GroupID:        sg.GroupId,
			ProjectNameTag: aws.FetchEc2Tag(sg.Tags, to.Strp("ProjectName")),
			ConfigNameTag:  aws.FetchEc2Tag(sg.Tags, to.Strp("ConfigName")),
			ServiceNameTag: aws.FetchEc2Tag(sg.Tags, to.Strp("ServiceName")),
		})
	}
	return sgs
}
