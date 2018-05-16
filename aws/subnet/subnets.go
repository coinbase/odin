package subnet

import (
	"fmt"

	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/coinbase/odin/aws"
	"github.com/coinbase/step/utils/to"
)

// Subnet struct
type Subnet struct {
	SubnetID      *string
	DeployWithTag *string
}

// Find returns a list of subnets for either ids or tags NO MIXING , e.g. subnet-00000000 OR privatea
func Find(ec2Client aws.EC2API, nameTagsOrIDs []*string) ([]*Subnet, error) {
	ids, tags := splitIDsTags(nameTagsOrIDs)

	subnets := []*Subnet{}

	if len(ids) > 0 {
		sns, err := findByID(ec2Client, ids)
		if err != nil {
			return nil, err
		}
		subnets = append(subnets, sns...)
	}

	if len(tags) > 0 {
		sns, err := findByTag(ec2Client, tags)
		if err != nil {
			return nil, err
		}
		subnets = append(subnets, sns...)
	}

	if len(subnets) != len(nameTagsOrIDs) {
		return nil, fmt.Errorf("Incorrect Number of Subnets Found. Found %v, Required %v", len(subnets), len(nameTagsOrIDs))
	}

	return subnets, nil
}

// isID sees if a string is
func isID(name string) bool {
	if len(name) < 8 {
		return false
	}

	return (name)[0:7] == "subnet-"
}

// splitIDsTags returns list of ids, and list of tags
func splitIDsTags(nameTagsOrIDs []*string) ([]*string, []*string) {
	ids := []*string{}
	tags := []*string{}
	for _, sn := range nameTagsOrIDs {
		if isID(*sn) {
			ids = append(ids, sn)
		} else {
			tags = append(tags, sn)
		}
	}

	return ids, tags
}

func findByID(ec2Client aws.EC2API, ids []*string) ([]*Subnet, error) {
	return find(ec2Client, &ec2.DescribeSubnetsInput{SubnetIds: ids})
}

func findByTag(ec2Client aws.EC2API, nameTags []*string) ([]*Subnet, error) {
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

	return find(ec2Client, &ec2.DescribeSubnetsInput{Filters: filters})
}

func find(ec2Client aws.EC2API, in *ec2.DescribeSubnetsInput) ([]*Subnet, error) {
	output, err := ec2Client.DescribeSubnets(in)

	if err != nil {
		return nil, err
	}

	subnets := []*Subnet{}
	for _, subnet := range output.Subnets {
		subnets = append(subnets, &Subnet{
			subnet.SubnetId,
			aws.FetchEc2Tag(subnet.Tags, to.Strp("DeployWith")),
		})
	}

	return subnets, nil
}
