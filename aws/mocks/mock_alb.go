package mocks

import (
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/coinbase/odin/aws"
	"github.com/coinbase/step/utils/to"
)

// ALBClient return
type ALBClient struct {
	aws.ALBAPI
	DescribeTargetGroupsResp map[string]*DescribeTargetGroupsResponse
	DescribeTagsResp         map[string]*DescribeV2TagsResponse
	DescribeTargetHealthResp map[string]*DescribeTargetHealthResponse
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
}

// AddTargetGroup return
func (m *ALBClient) AddTargetGroup(name string, projectName string, configName string, serviceName string) {
	m.init()
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
						&elbv2.Tag{Key: to.Strp("ProjectName"), Value: to.Strp(projectName)},
						&elbv2.Tag{Key: to.Strp("ConfigName"), Value: to.Strp(configName)},
						&elbv2.Tag{Key: to.Strp("ServiceName"), Value: to.Strp(serviceName)},
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
