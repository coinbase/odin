package mocks

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/coinbase/odin/aws"
	"github.com/coinbase/step/utils/to"
)

// GetInstanceProfileResponse returns
type GetInstanceProfileResponse struct {
	Resp  *iam.GetInstanceProfileOutput
	Error error
}

// GetRoleResponse returns
type GetRoleResponse struct {
	Resp  *iam.GetRoleOutput
	Error error
}

// IAMClient returns
type IAMClient struct {
	aws.IAMAPI
	GetInstanceProfileResp map[string]*GetInstanceProfileResponse
	GetRoleResp            map[string]*GetRoleResponse
}

func (m *IAMClient) init() {
	if m.GetInstanceProfileResp == nil {
		m.GetInstanceProfileResp = map[string]*GetInstanceProfileResponse{}
	}

	if m.GetRoleResp == nil {
		m.GetRoleResp = map[string]*GetRoleResponse{}
	}
}

// AWSProfileNotFoundError returns
func AWSProfileNotFoundError() error {
	return awserr.New(iam.ErrCodeNoSuchEntityException, "NoSuchEntity", nil)
}

// AddGetInstanceProfile returns
func (m *IAMClient) AddGetInstanceProfile(profileName string, path string) {
	m.init()
	m.GetInstanceProfileResp[profileName] = &GetInstanceProfileResponse{
		Resp: &iam.GetInstanceProfileOutput{
			InstanceProfile: &iam.InstanceProfile{
				Arn:  to.Strp(fmt.Sprintf("%v%v", path, profileName)),
				Path: to.Strp(path),
			},
		},
	}
}

// AddGetRole returns
func (m *IAMClient) AddGetRole(roleName string) {
	m.init()
	m.GetRoleResp[roleName] = &GetRoleResponse{
		Resp: &iam.GetRoleOutput{
			Role: &iam.Role{
				Arn: to.Strp(roleName),
			},
		},
	}
}

// GetInstanceProfile returns
func (m *IAMClient) GetInstanceProfile(in *iam.GetInstanceProfileInput) (*iam.GetInstanceProfileOutput, error) {
	m.init()
	resp := m.GetInstanceProfileResp[*in.InstanceProfileName]
	if resp == nil {
		return nil, AWSProfileNotFoundError()
	}
	return resp.Resp, resp.Error
}

// GetRole returns
func (m *IAMClient) GetRole(in *iam.GetRoleInput) (*iam.GetRoleOutput, error) {
	m.init()
	resp := m.GetRoleResp[*in.RoleName]
	if resp == nil {
		return nil, AWSProfileNotFoundError()
	}
	return resp.Resp, resp.Error
}
