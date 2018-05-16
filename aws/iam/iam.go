package iam

import (
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/coinbase/odin/aws"
)

//////
// PROFILE
//////

// Profile struct
type Profile struct {
	Path *string
	Arn  *string
}

// Find returns profile with name
func Find(iamClient aws.IAMAPI, profileName *string) (*Profile, error) {
	profileOutput, err := iamClient.GetInstanceProfile(&iam.GetInstanceProfileInput{
		InstanceProfileName: profileName,
	})

	if err != nil {
		return nil, err
	}

	awsProfile := profileOutput.InstanceProfile
	return &Profile{
		Path: awsProfile.Path,
		Arn:  awsProfile.Arn,
	}, nil
}

//////
// ROLE
//////

// RoleExists returns whether profile exists
func RoleExists(iamc aws.IAMAPI, roleName *string) error {
	_, err := iamc.GetRole(&iam.GetRoleInput{
		RoleName: roleName,
	})

	return err
}
