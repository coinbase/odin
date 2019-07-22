package ami

import (
	"fmt"

	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/coinbase/odin/aws"
	"github.com/coinbase/step/utils/to"
)

// Image struct
type Image struct {
	ImageID       *string
	DeployWithTag *string
}

func isID(name string) bool {
	if len(name) < 5 {
		return false
	}

	return (name)[0:4] == "ami-"
}

// Find takes either a ID or a Tag of an ami e.g. ubuntu or ami-00000000
func Find(ec2c aws.EC2API, nameTagOrID *string) (*Image, error) {
	if nameTagOrID == nil {
		return nil, fmt.Errorf("AMI Image nil")
	}
	if isID(*nameTagOrID) {
		return findByID(ec2c, nameTagOrID)
	}
	return findByTag(ec2c, nameTagOrID)
}

func findByID(ec2c aws.EC2API, id *string) (*Image, error) {
	return find(ec2c, &ec2.DescribeImagesInput{ImageIds: []*string{id}})
}

func findByTag(ec2c aws.EC2API, nameTag *string) (*Image, error) {
	filters := []*ec2.Filter{
		&ec2.Filter{
			Name:   to.Strp("tag:Name"),
			Values: []*string{nameTag},
		},
	}

	return find(ec2c, &ec2.DescribeImagesInput{Filters: filters})
}

func find(ec2c aws.EC2API, in *ec2.DescribeImagesInput) (*Image, error) {
	output, err := ec2c.DescribeImages(in)

	if err != nil {
		return nil, err
	}

	switch len(output.Images) {
	case 0:
		return nil, nil
	case 1:
		im := output.Images[0]
		if im == nil {
			return nil, fmt.Errorf("AMI Image nil")
		}
		return &Image{
			im.ImageId,
			aws.FetchEc2Tag(im.Tags, to.Strp("DeployWith")),
		}, nil
	default:
		return nil, fmt.Errorf("Must be exactly 1 Image with tag Name, there are %v", len(output.Images))
	}
}
