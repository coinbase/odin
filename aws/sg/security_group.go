package sg

import (
	"fmt"

	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/coinbase/odin/aws"
	"github.com/coinbase/step/utils/to"
)

// SecurityGroup struct
type SecurityGroup struct {
	NameTag        *string
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
	output, err := ec2Client.DescribeSecurityGroups(&ec2.DescribeSecurityGroupsInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{
				Name:   to.Strp("tag:Name"),
				Values: nameTags,
			}}})

	if err != nil {
		return nil, err
	}

	sgs := newSGs(output.SecurityGroups)

	// Need to validate that each Name tag matches Exactly one Security Group
	for _, nameTag := range nameTags {
		matches := 0
		for _, sg := range sgs {
			if sg.NameTag == nil {
				return nil, fmt.Errorf("SecurityGroup '%v': incorrect Name Tag", *nameTag)
			}

			if *sg.NameTag == *nameTag {
				matches += 1
			}
		}

		switch matches {
		case 0:
			return nil, fmt.Errorf("SecurityGroup '%v': not found", *nameTag)
		case 1:
			// Do nothing
		default:
			return nil, fmt.Errorf("SecurityGroup '%v': too many found", *nameTag)
		}
	}

	if len(sgs) != len(nameTags) {
		// Last assurance that no additional security groups were found
		return nil, fmt.Errorf("SecurityGroup: found %v required %v", len(sgs), len(nameTags))
	}

	return sgs, nil
}

func newSGs(output []*ec2.SecurityGroup) []*SecurityGroup {
	sgs := []*SecurityGroup{}

	for _, sg := range output {
		sgs = append(sgs, &SecurityGroup{
			GroupID:        sg.GroupId,
			NameTag:        aws.FetchEc2Tag(sg.Tags, to.Strp("Name")),
			ProjectNameTag: aws.FetchEc2Tag(sg.Tags, to.Strp("ProjectName")),
			ConfigNameTag:  aws.FetchEc2Tag(sg.Tags, to.Strp("ConfigName")),
			ServiceNameTag: aws.FetchEc2Tag(sg.Tags, to.Strp("ServiceName")),
		})
	}

	return sgs
}
