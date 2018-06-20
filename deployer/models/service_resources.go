package models

import (
	"fmt"

	"github.com/coinbase/odin/aws"
	"github.com/coinbase/odin/aws/alb"
	"github.com/coinbase/odin/aws/ami"
	"github.com/coinbase/odin/aws/asg"
	"github.com/coinbase/odin/aws/elb"
	"github.com/coinbase/odin/aws/iam"
	"github.com/coinbase/odin/aws/sg"
	"github.com/coinbase/odin/aws/subnet"
	"github.com/coinbase/step/utils/is"
	"github.com/coinbase/step/utils/to"
)

// This models all resources referenced for a release
type serviceIface interface {
	ProjectName() *string
	ConfigName() *string
	Name() *string
	ReleaseID() *string
}

type pcsresourceIface interface {
	ProjectName() *string
	ConfigName() *string
	ServiceName() *string
}

// ServiceResources struct
type ServiceResources struct {
	Image          *ami.Image
	Profile        *iam.Profile
	PrevASG        *asg.ASG
	SecurityGroups []*sg.SecurityGroup
	ELBs           []*elb.LoadBalancer
	TargetGroups   []*alb.TargetGroup
	Subnets        []*subnet.Subnet
}

// ServiceResourceNames struct
type ServiceResourceNames struct {
	Image          *string   `json:"image,omitempty"`
	Profile        *string   `json:"profile_arn,omitempty"`
	PrevASG        *string   `json:"prev_asg_arn,omitempty"`
	SecurityGroups []*string `json:"security_groups,omitempty"`
	ELBs           []*string `json:"elbs,omitempty"`
	TargetGroups   []*string `json:"target_group_arns,omitempty"`
	Subnets        []*string `json:"subnets,omitempty"`
}

// ToServiceResourceNames returns
func (sr *ServiceResources) ToServiceResourceNames() *ServiceResourceNames {
	var im *string
	if sr.Image != nil {
		im = sr.Image.ImageID
	}

	var profile *string
	if sr.Profile != nil {
		profile = sr.Profile.Arn
	}

	var prevASG *string
	if sr.PrevASG != nil {
		prevASG = sr.PrevASG.AutoScalingGroupName
	}

	sgs := []*string{}
	for _, sg := range sr.SecurityGroups {
		if sg == nil || is.EmptyStr(sg.GroupID) {
			continue
		}

		sgs = append(sgs, sg.GroupID)
	}

	elbs := []*string{}
	for _, elb := range sr.ELBs {
		if elb == nil || is.EmptyStr(elb.LoadBalancerName) {
			continue
		}

		elbs = append(elbs, elb.LoadBalancerName)
	}

	tgs := []*string{}
	for _, tg := range sr.TargetGroups {
		if tg == nil || is.EmptyStr(tg.TargetGroupArn) {
			continue
		}

		tgs = append(tgs, tg.TargetGroupArn)
	}

	subnets := []*string{}
	for _, subnet := range sr.Subnets {
		if subnet == nil || is.EmptyStr(subnet.SubnetID) {
			continue
		}

		subnets = append(subnets, subnet.SubnetID)
	}

	return &ServiceResourceNames{
		Image:          im,
		Profile:        profile,
		PrevASG:        prevASG,
		SecurityGroups: sgs,
		ELBs:           elbs,
		TargetGroups:   tgs,
		Subnets:        subnets,
	}
}

// Validate returns
func (sr *ServiceResources) Validate(service *Service) error {

	if err := sr.validateAttributes(service); err != nil {
		return err
	}

	if err := ValidateImage(service, sr.Image); err != nil {
		return err
	}

	// Now the Easy Validations are over time to validate Tags and Paths
	if err := ValidateIAMProfile(service, sr.Profile); err != nil {
		return err
	}

	if err := ValidatePrevASG(service, sr.PrevASG); err != nil {
		return err
	}

	for _, r := range sr.Subnets {
		if err := ValidateSubnet(service, r); err != nil {
			return err
		}
	}

	for _, r := range sr.SecurityGroups {
		if err := ValidateSecurityGroup(service, r); err != nil {
			return err
		}
	}

	for _, r := range sr.ELBs {
		if err := ValidateELB(service, r); err != nil {
			return err
		}
	}

	for _, r := range sr.TargetGroups {
		if err := ValidateTargetGroup(service, r); err != nil {
			return err
		}
	}

	return nil
}

func (sr *ServiceResources) validateAttributes(service *Service) error {
	names := sr.ToServiceResourceNames()

	// Must have Image
	if sr.Image == nil {
		return fmt.Errorf("Image is nil")
	}

	// Must have the correct amount of security groups, ELBS and Target Groups
	if len(service.SecurityGroups) != len(sr.SecurityGroups) {
		return fmt.Errorf("Security Group Not Found actual %v expected %v", to.StrSlice(names.SecurityGroups), to.StrSlice(service.SecurityGroups))
	}

	if len(service.ELBs) != len(sr.ELBs) {
		return fmt.Errorf("ELB Not Found actual %v expected %v", to.StrSlice(names.ELBs), to.StrSlice(service.ELBs))
	}

	if len(service.TargetGroups) != len(sr.TargetGroups) {
		return fmt.Errorf("TargetGroup Not Found actual %v expected %v", to.StrSlice(names.TargetGroups), to.StrSlice(service.TargetGroups))
	}

	if len(service.Subnets()) != len(sr.Subnets) {
		return fmt.Errorf("Subnets Not Found actual %v expected %v", to.StrSlice(names.Subnets), to.StrSlice(service.Subnets()))
	}

	return nil
}

// ValidateImage returns
func ValidateImage(service serviceIface, im *ami.Image) error {
	if im == nil {
		return fmt.Errorf("Image is nil")
	}

	if im.DeployWithTag == nil {
		return fmt.Errorf("Image %v DeployWith Tag nil", *im.ImageID)
	}

	if *im.DeployWithTag != "odin" {
		return fmt.Errorf("Image %v DeployWith Tag expected: %v actual: %v", *im.ImageID, "odin", *im.DeployWithTag)
	}

	return nil
}

// ValidateSubnet returns
func ValidateSubnet(service serviceIface, subnet *subnet.Subnet) error {
	if subnet == nil {
		return fmt.Errorf("Subnet is nil")
	}

	if subnet.DeployWithTag == nil {
		return fmt.Errorf("Subnet %v DeployWith Tag nil", *subnet.SubnetID)
	}

	if *subnet.DeployWithTag != "odin" {
		return fmt.Errorf("Subnet %v DeployWith Tag expected: %v actual: %v", *subnet.SubnetID, "odin", *subnet.DeployWithTag)
	}

	return nil
}

// ValidatePrevASG returns
func ValidatePrevASG(service serviceIface, as *asg.ASG) error {
	if as == nil {
		return nil // Allowed to not have previous ASG
	}

	// None of these should happen but it is just being extra safe
	if !aws.HasProjectName(as, service.ProjectName()) {
		return fmt.Errorf("Previous ASG incorrect ProjectName requires %q has %q", to.Strs(service.ProjectName()), to.Strs(as.ProjectName()))
	}

	if !aws.HasConfigName(as, service.ConfigName()) {
		return fmt.Errorf("Previous ASG incorrect ConfigName requires %q has %q", to.Strs(service.ConfigName()), to.Strs(as.ConfigName()))
	}

	if !aws.HasServiceName(as, service.Name()) {
		return fmt.Errorf("Previous ASG incorrect ServiceName requires %q has %q", to.Strs(service.Name()), to.Strs(as.ServiceName()))
	}

	if as.ReleaseID() == nil {
		return fmt.Errorf("Previous ASG ReleaseID nil")
	}

	// Existing ASG must not have the same Release ID
	if *as.ReleaseID() == *service.ReleaseID() {
		return fmt.Errorf("Previous ASG incorrect ReleaseID requires %q has %q", to.Strs(service.ReleaseID()), to.Strs(as.ReleaseID()))
	}

	return nil
}

// ValidateIAMProfile returns
func ValidateIAMProfile(service serviceIface, profile *iam.Profile) error {
	if profile == nil {
		return nil // Profile is allowed to be nil
	}

	if profile.Path == nil {
		// Again should never happen
		return fmt.Errorf("Iam Profile Path not found")
	}

	// This allows for default profiles for all services || all configs || all projects
	specificPath := fmt.Sprintf("/%v/%v/%v/", *service.ProjectName(), *service.ConfigName(), *service.Name())
	validPaths := []string{
		specificPath,
		fmt.Sprintf("/%v/%v/%v/", *service.ProjectName(), *service.ConfigName(), "_all"),
		fmt.Sprintf("/%v/%v/%v/", *service.ProjectName(), "_all", "_all"),
		fmt.Sprintf("/%v/%v/%v/", "_all", "_all", "_all"),
	}

	for _, validPath := range validPaths {
		if *profile.Path == validPath {
			return nil
		}
	}

	// Again should never happen
	return fmt.Errorf("Iam Profile Path incorrect, it is %q and requires %q", *profile.Path, specificPath)
}

// ValidateSecurityGroup returns
func ValidateSecurityGroup(service serviceIface, sc *sg.SecurityGroup) error {
	return validateProjectConfigServiceNames("SecurityGroup", service, sc)
}

// ValidateELB returns
func ValidateELB(service serviceIface, lb *elb.LoadBalancer) error {
	return validateProjectConfigServiceNames("ELB", service, lb)
}

// ValidateTargetGroup returns
func ValidateTargetGroup(service serviceIface, tg *alb.TargetGroup) error {
	return validateProjectConfigServiceNames("TargetGroup", service, tg)
}

func validateProjectConfigServiceNames(prefix string, service serviceIface, r pcsresourceIface) error {
	if r == nil {
		return fmt.Errorf("%v is nil", prefix)
	}

	if !aws.HasProjectName(r, service.ProjectName()) && !aws.HasAllValue(r.ProjectName()) {
		return fmt.Errorf("%v incorrect ProjectName requires %q has %q", prefix, to.Strs(service.ProjectName()), to.Strs(r.ProjectName()))
	}

	if !aws.HasConfigName(r, service.ConfigName()) && !aws.HasAllValue(r.ConfigName()) {
		return fmt.Errorf("%v incorrect ConfigName requires %q has %q", prefix, to.Strs(service.ConfigName()), to.Strs(r.ConfigName()))
	}

	if !aws.HasServiceName(r, service.Name()) && !aws.HasAllValue(r.ServiceName()) {
		return fmt.Errorf("%v incorrect ServiceName requires %q has %q", prefix, to.Strs(service.Name()), to.Strs(r.ServiceName()))
	}

	return nil
}
