package mocks

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/coinbase/odin/aws"
	"github.com/coinbase/step/utils/to"
)

// ALBClient return
type ALBClient struct {
	aws.ALBAPI
	DescribeTargetGroupsResp          map[string]*DescribeTargetGroupsResponse
	DescribeTagsResp                  map[string]*DescribeV2TagsResponse
	DescribeTargetHealthResp          map[string]*DescribeTargetHealthResponse
	DescribeTargetGroupAttributesResp map[string]*DescribeTargetGroupAttributesResponse
}

// DescribeTargetGroupsResponse return
type DescribeTargetGroupsResponse struct {
	Resp  *elbv2.DescribeTargetGroupsOutput
	Error error
}

// DescribeV2TagsResponse return
type DescribeV2TagsResponse struct {
	Resp  *elbv2.DescribeTagsOutput
	Error error
}

// DescribeTargetHealthResponse return
type DescribeTargetHealthResponse struct {
	Resp  *elbv2.DescribeTargetHealthOutput
	Error error
}

// DescribeTargetGroupAttributesResponse return
type DescribeTargetGroupAttributesResponse struct {
	Resp  *elbv2.DescribeTargetGroupAttributesOutput
	Error error
}

// MockTargetGroup configuration struct, with defaults
type MockTargetGroup struct {
	Name           string
	ProjectName    string
	ConfigName     string
	ServiceName    string
	AllowedService string
}

func (tg MockTargetGroup) allowedService() string {
	if tg.AllowedService == "" {
		return fmt.Sprintf("%s::%s::%s", tg.ProjectName, tg.ConfigName, tg.ServiceName)
	}
	return tg.AllowedService
}

func (tg *MockTargetGroup) init() {
	if tg.Name == "" {
		tg.Name = "tg_name"
	}
	if tg.ProjectName == "" {
		tg.ProjectName = "project_name"
	}
	if tg.ConfigName == "" {
		tg.ConfigName = "config_name"
	}
	if tg.ServiceName == "" {
		tg.ServiceName = "service_name"
	}
}

// AWSTargetGroupNotFoundError return
func AWSTargetGroupNotFoundError() error {
	return awserr.New(elbv2.ErrCodeTargetGroupNotFoundException, "TargetGroupNotFound", nil)
}

func (m *ALBClient) init() {
	if m.DescribeTargetGroupsResp == nil {
		m.DescribeTargetGroupsResp = map[string]*DescribeTargetGroupsResponse{}
	}

	if m.DescribeTagsResp == nil {
		m.DescribeTagsResp = map[string]*DescribeV2TagsResponse{}
	}

	if m.DescribeTargetHealthResp == nil {
		m.DescribeTargetHealthResp = map[string]*DescribeTargetHealthResponse{}
	}

	if m.DescribeTargetGroupAttributesResp == nil {
		m.DescribeTargetGroupAttributesResp = map[string]*DescribeTargetGroupAttributesResponse{}
	}
}

// AddTargetGroup return
func (m *ALBClient) AddTargetGroup(parameters MockTargetGroup) {
	m.init()
	parameters.init()

	name := parameters.Name
	m.DescribeTargetGroupsResp[name] = &DescribeTargetGroupsResponse{
		Resp: &elbv2.DescribeTargetGroupsOutput{
			TargetGroups: []*elbv2.TargetGroup{
				&elbv2.TargetGroup{TargetGroupName: &name, TargetGroupArn: &name},
			},
		},
	}

	m.DescribeTagsResp[name] = &DescribeV2TagsResponse{
		Resp: &elbv2.DescribeTagsOutput{
			TagDescriptions: []*elbv2.TagDescription{
				&elbv2.TagDescription{
					ResourceArn: &name,
					Tags: []*elbv2.Tag{
						&elbv2.Tag{Key: to.Strp("ProjectName"), Value: to.Strp(parameters.ProjectName)},
						&elbv2.Tag{Key: to.Strp("ConfigName"), Value: to.Strp(parameters.ConfigName)},
						&elbv2.Tag{Key: to.Strp("ServiceName"), Value: to.Strp(parameters.ServiceName)},
						&elbv2.Tag{Key: to.Strp("AllowedService"), Value: to.Strp(parameters.allowedService())},
					},
				},
			},
		},
	}

	m.DescribeTargetHealthResp[name] = &DescribeTargetHealthResponse{
		Resp: &elbv2.DescribeTargetHealthOutput{
			TargetHealthDescriptions: []*elbv2.TargetHealthDescription{
				&elbv2.TargetHealthDescription{
					Target:       &elbv2.TargetDescription{Id: to.Strp("InstanceId1")},
					TargetHealth: &elbv2.TargetHealth{State: to.Strp("healthy")},
				},
			},
		},
	}

	m.DescribeTargetGroupAttributesResp[name] = &DescribeTargetGroupAttributesResponse{
		Resp: &elbv2.DescribeTargetGroupAttributesOutput{
			Attributes: []*elbv2.TargetGroupAttribute{
				&elbv2.TargetGroupAttribute{
					Key:   to.Strp("slow_start.duration_seconds"),
					Value: to.Strp("42"),
				},
			},
		},
	}
}

// DescribeTargetGroups return
func (m *ALBClient) DescribeTargetGroups(in *elbv2.DescribeTargetGroupsInput) (*elbv2.DescribeTargetGroupsOutput, error) {
	m.init()
	lbName := in.Names[0]
	resp := m.DescribeTargetGroupsResp[*lbName]
	if resp == nil {
		return nil, AWSTargetGroupNotFoundError()
	}
	return resp.Resp, resp.Error
}

// DescribeTags return
func (m *ALBClient) DescribeTags(in *elbv2.DescribeTagsInput) (*elbv2.DescribeTagsOutput, error) {
	m.init()
	lbName := in.ResourceArns[0]
	resp := m.DescribeTagsResp[*lbName]
	if resp == nil {
		return nil, AWSTargetGroupNotFoundError()
	}
	return resp.Resp, resp.Error
}

// DescribeTargetHealth return
func (m *ALBClient) DescribeTargetHealth(in *elbv2.DescribeTargetHealthInput) (*elbv2.DescribeTargetHealthOutput, error) {
	m.init()
	lbName := in.TargetGroupArn
	resp := m.DescribeTargetHealthResp[*lbName]
	if resp == nil {
		return nil, AWSTargetGroupNotFoundError()
	}

	if resp.Resp == nil {
		return &elbv2.DescribeTargetHealthOutput{}, nil
	}

	return resp.Resp, resp.Error
}

// DescribeTargetGroupAttributes return
func (m *ALBClient) DescribeTargetGroupAttributes(in *elbv2.DescribeTargetGroupAttributesInput) (*elbv2.DescribeTargetGroupAttributesOutput, error) {
	m.init()
	arn := in.TargetGroupArn
	resp := m.DescribeTargetGroupAttributesResp[*arn]
	if resp == nil {
		return nil, AWSTargetGroupNotFoundError()
	}
	return resp.Resp, resp.Error
}
