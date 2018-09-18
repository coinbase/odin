package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Release_FetchResources_Works(t *testing.T) {
	// func (release *Release) FetchResources(asgc aws.ASGAPI, ec2 aws.EC2API, elbc aws.ELBAPI, albc aws.ALBAPI, iamc aws.IAMAPI, snsc aws.SNSAPI) (map[string]*ServiceResources, error)
	r := MockRelease(t)
	MockPrepareRelease(r)

	awsc := MockAwsClients(r)

	sm, err := r.FetchResources(awsc.ASG, awsc.EC2, awsc.ELB, awsc.ALB, awsc.IAM, awsc.SNS)
	assert.NoError(t, err)

	assert.Equal(t, 1, len(sm))
}

func Test_Release_ValidateResources_Works(t *testing.T) {
	// func (release *Release) ValidateResources(resources map[string]*ServiceResources) error {
	r := MockRelease(t)
	MockPrepareRelease(r)

	awsc := MockAwsClients(r)

	sm, err := r.FetchResources(awsc.ASG, awsc.EC2, awsc.ELB, awsc.ALB, awsc.IAM, awsc.SNS)
	assert.NoError(t, err)

	assert.NoError(t, r.ValidateResources(sm))
}

func Test_Release_UpdateWithResources_Works(t *testing.T) {
	// func (release *Release) UpdateWithResources(resources map[string]*ServiceResources) {
	r := MockRelease(t)
	MockPrepareRelease(r)

	awsc := MockAwsClients(r)

	sm, err := r.FetchResources(awsc.ASG, awsc.EC2, awsc.ELB, awsc.ALB, awsc.IAM, awsc.SNS)
	assert.NoError(t, err)

	r.UpdateWithResources(sm)
}

func Test_Release_CreateResources_Works(t *testing.T) {
	// func (release *Release) CreateResources(asgc aws.ASGAPI, cwc aws.CWAPI) error {
	r := MockRelease(t)
	MockPrepareRelease(r)

	awsc := MockAwsClients(r)
	assert.NoError(t, r.CreateResources(awsc.ASG, awsc.CW))
}

func Test_Release_UpdateHealthy_Works(t *testing.T) {
	// func (release *Release) UpdateHealthy(asgc aws.ASGAPI, elbc aws.ELBAPI, albc aws.ALBAPI) error {
	r := MockRelease(t)
	MockPrepareRelease(r)

	awsc := MockAwsClients(r)

	assert.NoError(t, r.CreateResources(awsc.ASG, awsc.CW))
	assert.NoError(t, r.UpdateHealthy(awsc.ASG, awsc.ELB, awsc.ALB))
}

func Test_Release_SuccessfulTearDown_Works(t *testing.T) {
	// func (release *Release) SuccessfulTearDown(asgc aws.ASGAPI, cwc aws.CWAPI) error {
	r := MockRelease(t)
	MockPrepareRelease(r)

	awsc := MockAwsClients(r)
	assert.NoError(t, r.SuccessfulTearDown(awsc.ASG, awsc.CW))
}

func Test_Release_UnsuccessfulTearDown_Works(t *testing.T) {
	// func (release *Release) UnsuccessfulTearDown(asgc aws.ASGAPI, cwc aws.CWAPI) error {
	r := MockRelease(t)
	MockPrepareRelease(r)

	awsc := MockAwsClients(r)
	assert.NoError(t, r.UnsuccessfulTearDown(awsc.ASG, awsc.CW))
}
