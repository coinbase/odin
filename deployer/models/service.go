package models

import (
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/autoscaling/autoscalingiface"
	"github.com/coinbase/odin/aws"
	"github.com/coinbase/odin/aws/alb"
	"github.com/coinbase/odin/aws/asg"
	"github.com/coinbase/odin/aws/elb"
	"github.com/coinbase/odin/aws/iam"
	"github.com/coinbase/odin/aws/lc"
	"github.com/coinbase/odin/aws/sg"
	"github.com/coinbase/step/utils/is"
	"github.com/coinbase/step/utils/to"
)

// HealthReport is built to make log lines like:
// web: .....|.
// gray targets, red terminated, yellow unhealthy, green healthy
type HealthReport struct {
	TargetHealthy  *int     `json:"target_healthy,omitempty"`  // Number of instances aimed to to Launch
	TargetLaunched *int     `json:"target_launched,omitempty"` // Number of instances aimed to to Launch
	Healthy        *int     `json:"healthy,omitempty"`         // Number of instances that are healthy
	Launching      *int     `json:"launching,omitempty"`       // Number of instances that have been created
	Terminating    *int     `json:"terminating,omitempty"`     // Number of instances that are Terminating
	TerminatingIDs []string `json:"terminating_ids,omitempty"` // Instance IDs that are Terminating
}

// TYPES

// Service struct
type Service struct {
	release  *Release
	userdata *string

	// Generated
	ServiceName *string `json:"service_name,omitempty"`

	// Find these Resources
	ELBs           []*string          `json:"elbs,omitempty"`
	Profile        *string            `json:"profile,omitempty"`
	TargetGroups   []*string          `json:"target_groups,omitempty"`
	SecurityGroups []*string          `json:"security_groups,omitempty"`
	Tags           map[string]*string `json:"tags,omitempty"`

	// Create Resources
	InstanceType *string            `json:"instance_type,omitempty"`
	Autoscaling  *AutoScalingConfig `json:"autoscaling,omitempty"`

	// EBS
	EBSVolumeSize *int64  `json:"ebs_volume_size,omitempty"`
	EBSVolumeType *string `json:"ebs_volume_type,omitempty"`
	EBSDeviceName *string `json:"ebs_device_name,omitempty"`

	// Found Resources
	Resources *ServiceResourceNames `json:"resources,omitempty"`

	// Created Resources
	CreatedASG              *string `json:"created_asg,omitempty"`
	PreviousDesiredCapacity *int64  `json:"previous_desired_capacity,omitempty"`

	// What is Healthy
	HealthReport *HealthReport `json:"healthy_report,omitempty"`
	Healthy      bool
}

//////////
// Getters
//////////

// ProjectName returns project name
func (service *Service) ProjectName() *string {
	return service.release.ProjectName
}

// ConfigName returns config name
func (service *Service) ConfigName() *string {
	return service.release.ConfigName
}

// Name service name
func (service *Service) Name() *string {
	return service.ServiceName
}

// ReleaseUUID returns release UUID
func (service *Service) ReleaseUUID() *string {
	return service.release.UUID
}

// ReleaseID returns release ID
func (service *Service) ReleaseID() *string {
	return service.release.ReleaseID
}

// CreatedAt returns created at data
func (service *Service) CreatedAt() *time.Time {
	return service.release.CreatedAt
}

// ServiceID returns a formatted string of the services ID
func (service *Service) ServiceID() *string {
	if service.ProjectName() == nil || service.ConfigName() == nil || service.ServiceName == nil || service.CreatedAt() == nil {
		return nil
	}

	tf := strings.Replace(service.CreatedAt().UTC().Format(time.RFC3339), ":", "-", -1)
	return to.Strp(fmt.Sprintf("%v-%v-%v-%v", *service.ProjectName(), *service.ConfigName(), tf, *service.ServiceName))
}

// Subnets returns subnets
func (service *Service) Subnets() []*string {
	return service.release.Subnets
}

// UserData will take the releases template and override
func (service *Service) UserData() *string {
	templateARGs := []string{}
	templateARGs = append(templateARGs, "{{RELEASE_ID}}", to.Strs(service.ReleaseID()))
	templateARGs = append(templateARGs, "{{PROJECT_NAME}}", to.Strs(service.ProjectName()))
	templateARGs = append(templateARGs, "{{CONFIG_NAME}}", to.Strs(service.ConfigName()))
	templateARGs = append(templateARGs, "{{SERVICE_NAME}}", to.Strs(service.ServiceName))

	replacer := strings.NewReplacer(templateARGs...)

	return to.Strp(replacer.Replace(to.Strs(service.userdata)))
}

// SetUserData sets the userdata
func (service *Service) SetUserData(userdata *string) {
	service.userdata = userdata
}

// LifeCycleHooks returns
func (service *Service) LifeCycleHooks() map[string]*LifeCycleHook {
	return service.release.LifeCycleHooks
}

// SubnetIds returns
func (service *Service) SubnetIds() *string {
	return to.Strp(strings.Join(to.StrSlice(service.Resources.Subnets), ","))
}

// LifeCycleHookSpecs returns
func (service *Service) LifeCycleHookSpecs() []*autoscaling.LifecycleHookSpecification {
	lcs := []*autoscaling.LifecycleHookSpecification{}
	for _, lc := range service.LifeCycleHooks() {
		lcs = append(lcs, lc.ToLifecycleHookSpecification())
	}
	return lcs
}

func (service *Service) targetCapacity() int {
	return service.Autoscaling.TargetCapacity(service.PreviousDesiredCapacity)
}

func (service *Service) target() int {
	return service.Autoscaling.TargetHealthy(service.PreviousDesiredCapacity)
}

func (service *Service) maxTerminations() int {
	return service.Autoscaling.MaxTerminationsInt()
}

func (service *Service) errorPrefix() string {
	if service.ServiceName == nil {
		return fmt.Sprintf("Service Error:")
	}
	return fmt.Sprintf("Service(%v) Error:", *service.ServiceName)
}

//////////
// Setters
//////////

// SetDefaults assigns default values
func (service *Service) SetDefaults(release *Release, serviceName string) {
	service.release = release

	service.ServiceName = &serviceName

	// Autoscaling Defaults
	if service.Autoscaling == nil {
		service.Autoscaling = &AutoScalingConfig{}
	}

	if service.Resources == nil {
		service.Resources = &ServiceResourceNames{
			Subnets: []*string{to.Strp("place_holder")},
		}
	}

	service.Autoscaling.SetDefaults(service.ServiceID())
}

// setHealthy sets the health state from the instances
func (service *Service) setHealthy(instances aws.Instances) {
	healthy := instances.HealthyIDs()
	terming := instances.TerminatingIDs()

	service.HealthReport = &HealthReport{
		TargetHealthy:  to.Intp(service.target()),
		TargetLaunched: to.Intp(service.targetCapacity()),
		Healthy:        to.Intp(len(healthy)),
		Terminating:    to.Intp(len(terming)),
		TerminatingIDs: terming,
		Launching:      to.Intp(len(instances)),
	}

	// The Service is Healthy if
	// the number of instances that are healthy is greater than or equal to the target
	service.Healthy = len(healthy) >= service.target()
}

//////////
// Validate
//////////

// Validate validates the service
func (service *Service) Validate() error {
	if err := service.ValidateAttributes(); err != nil {
		return fmt.Errorf("%v %v", service.errorPrefix(), err.Error())
	}

	for name, lc := range service.LifeCycleHooks() {
		if lc == nil {
			return fmt.Errorf("LifeCycle %v is nil", name)
		}

		err := lc.ValidateAttributes()
		if err != nil {
			return err
		}
	}

	// VALIDATE Autoscaling Group Input (this in implemented by AWS)
	if err := service.createInput().Validate(); err != nil {
		return fmt.Errorf("%v %v", service.errorPrefix(), err.Error())
	}

	if err := service.createLaunchConfigurationInput().Validate(); err != nil {
		return fmt.Errorf("%v %v", service.errorPrefix(), err.Error())
	}

	return nil
}

// ValidateAttributes validates attributes
func (service *Service) ValidateAttributes() error {
	if is.EmptyStr(service.ServiceName) {
		return fmt.Errorf("ServiceName must be defined")
	}

	if is.EmptyStr(service.InstanceType) {
		return fmt.Errorf("InstanceType must be defined")
	}

	if service.Autoscaling == nil {
		return fmt.Errorf("Autoscaling must be defined")
	}

	if err := service.Autoscaling.ValidateAttributes(); err != nil {
		return err
	}

	// Must have security groups
	if len(service.SecurityGroups) < 1 {
		return fmt.Errorf("Security Groups must be included")
	}

	if !is.UniqueStrp(service.SecurityGroups) {
		return fmt.Errorf("Security Group must be unique")
	}

	if !is.UniqueStrp(service.ELBs) {
		// Non unique string in ELBs or nil value
		return fmt.Errorf("Non Unique ELBs")
	}

	if !is.UniqueStrp(service.TargetGroups) {
		// Non unique string in ELBs or nil value
		return fmt.Errorf("Non Unique TargetGroups")
	}

	return nil
}

//////////
// Validate Resources
//////////

// FetchResources attempts to retrieve all resources
func (service *Service) FetchResources(ec2 aws.EC2API, elbc aws.ELBAPI, albc aws.ALBAPI, iamc aws.IAMAPI) (*ServiceResources, error) {
	// RESOURCES THAT ARE PROJECT-CONFIG-SERVICE specific
	// Fetch Security Group
	sgs, err := sg.Find(ec2, service.SecurityGroups)
	if err != nil {
		return nil, err
	}

	// FETCH ELBS
	elbs, err := elb.FindAll(elbc, service.ELBs)
	if err != nil {
		return nil, err
	}

	// Fetch TargetGroups
	targetGroups, err := alb.FindAll(albc, service.TargetGroups)
	if err != nil {
		return nil, err
	}

	// FETCH IAM
	var iamProfile *iam.Profile
	if service.Profile != nil {
		iamProfile, err = iam.Find(iamc, service.Profile)
		if err != nil {
			return nil, err
		}
	}

	return &ServiceResources{
		SecurityGroups: sgs,
		ELBs:           elbs,
		TargetGroups:   targetGroups,
		Profile:        iamProfile,
	}, nil
}

//////////
// Create Resources
//////////

// CreateResources creates the ASG and Launch configuration for the service
func (service *Service) CreateResources(asgc aws.ASGAPI, cwc aws.CWAPI) error {

	err := service.createLaunchConfiguration(asgc)
	if err != nil {
		return err
	}

	createdASG, err := service.createASG(asgc)

	if err != nil {
		return err
	}

	service.CreatedASG = createdASG.AutoScalingGroupName

	if err := service.createAutoScalingPolicies(asgc, cwc); err != nil {
		return err
	}

	service.setHealthy(aws.Instances{})
	return nil
}

func (service *Service) createInput() *asg.Input {
	input := &asg.Input{&autoscaling.CreateAutoScalingGroupInput{}}

	input.AutoScalingGroupName = service.ServiceID()
	input.LaunchConfigurationName = service.ServiceID()

	input.MinSize = service.Autoscaling.MinSize
	input.MaxSize = service.Autoscaling.MaxSize

	input.DefaultCooldown = service.Autoscaling.DefaultCooldown
	input.HealthCheckGracePeriod = service.Autoscaling.HealthCheckGracePeriod

	input.DesiredCapacity = to.Int64p(int64(service.targetCapacity()))

	input.LoadBalancerNames = service.Resources.ELBs
	input.TargetGroupARNs = service.Resources.TargetGroups

	input.VPCZoneIdentifier = service.SubnetIds()
	input.LifecycleHookSpecificationList = service.LifeCycleHookSpecs()

	for key, value := range service.Tags {
		input.AddTag(key, value)
	}

	input.AddTag("ProjectName", service.ProjectName())
	input.AddTag("ConfigName", service.ConfigName())
	input.AddTag("ServiceName", service.ServiceName)
	input.AddTag("ReleaseID", service.ReleaseID())
	input.AddTag("ReleaseUUID", service.ReleaseUUID())
	input.AddTag("Name", service.ServiceID())

	input.SetDefaults()

	return input
}

func (service *Service) createAutoScalingPolicies(asgc aws.ASGAPI, cwc aws.CWAPI) error {
	for _, policy := range service.Autoscaling.Policies {
		if err := policy.Create(asgc, cwc, service.ServiceID()); err != nil {
			return err
		}
	}

	return nil
}

func (service *Service) createASG(asgc aws.ASGAPI) (*asg.ASG, error) {
	input := service.createInput()

	if err := input.Create(asgc); err != nil {
		return nil, err
	}

	return input.ToASG(), nil
}

func (service *Service) createLaunchConfigurationInput() *lc.LaunchConfigInput {
	input := &lc.LaunchConfigInput{&autoscaling.CreateLaunchConfigurationInput{}}
	input.SetDefaults()

	input.LaunchConfigurationName = service.ServiceID()

	if service.Resources != nil {
		input.ImageId = service.Resources.Image
		input.SecurityGroups = service.Resources.SecurityGroups
		input.IamInstanceProfile = service.Resources.Profile
	}
	input.InstanceType = service.InstanceType

	input.UserData = to.Base64p(service.UserData())

	input.AddBlockDevice(service.EBSVolumeSize, service.EBSVolumeType, service.EBSDeviceName)

	return input
}

func (service *Service) createLaunchConfiguration(asgc autoscalingiface.AutoScalingAPI) error {
	input := service.createLaunchConfigurationInput()

	if err := input.Create(asgc); err != nil {
		return err
	}

	return nil
}

//////////
// Healthy Resources
//////////

// HaltError error
type HaltError struct {
	err error
}

// Error returns error
func (he *HaltError) Error() string {
	return he.err.Error()
}

// UpdateHealthy updates the health status of the service
// This might cause a Halt Error which will force the release to stop
func (service *Service) UpdateHealthy(asgc aws.ASGAPI, elbc aws.ELBAPI, albc aws.ALBAPI) error {
	all, err := asg.GetInstances(asgc, service.CreatedASG)
	if err != nil {
		return err // This might retry
	}

	// Early exit and Halt if there are instances Terminating
	if terming := all.TerminatingIDs(); len(terming) > service.maxTerminations() {
		err := fmt.Errorf("Found terming instances %v, %v", *service.ServiceName, strings.Join(terming, ","))
		return &HaltError{err} // This will immediately stop deploying
	}

	// Fetch All the instances
	for _, checkELB := range service.Resources.ELBs {
		elbInstances, err := elb.GetInstances(elbc, checkELB, all.InstanceIDs())
		if err != nil {
			return err // This might retry
		}

		all = all.MergeInstances(elbInstances)
	}

	for _, checkTG := range service.Resources.TargetGroups {
		tgInstances, err := alb.GetInstances(albc, checkTG, all.InstanceIDs())

		if err != nil {
			return err // This might retry
		}

		all = all.MergeInstances(tgInstances)
	}

	service.setHealthy(all)
	return nil
}
