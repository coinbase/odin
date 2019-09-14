package models

import (
	"fmt"

	"github.com/coinbase/step/aws"
	"github.com/coinbase/step/aws/s3"
	"github.com/coinbase/step/bifrost"
	"github.com/coinbase/step/utils/to"
)

//////////
// Safe Deploy
//////////

// ValidateSafeRelease will error if the currently deployed release has different:
// 1. Subnets, Image, or Services
// Or any service has different:
// 2. Security Groups or Profile
// 3. ELBs or Target Groups
// 4. Instance Type or Autoscaling Preferences
// 5. EBS information
// 6. AssociatePublicIpAddress
func (release *Release) ValidateSafeRelease(s3c aws.S3API, resources *ReleaseResources) error {
	if len(resources.PreviousASGs) == 0 {
		// If there are no currently deployed ASGs then we can ignore this check
		return nil
	}

	// Scaffold Previous Release
	previousRelease := Release{
		Release: bifrost.Release{
			ReleaseID:    resources.PreviousReleaseID,
			ProjectName:  release.ProjectName,
			ConfigName:   release.ConfigName,
			AwsAccountID: release.AwsAccountID,
			AwsRegion:    release.AwsRegion,
			Bucket:       release.Bucket,
		},
	}

	// Get Previous Release From S3
	err := s3.GetStruct(
		s3c,
		previousRelease.Bucket,
		previousRelease.ReleasePath(),
		&previousRelease,
	)

	if err != nil {
		switch err.(type) {
		case *s3.NotFoundError:
			// No lock to release
			return fmt.Errorf("SafeRelease Error: Cannot find previous release s3://%v/%v", *previousRelease.Bucket, *previousRelease.ReleasePath())
		default:
			return err // All other errors return
		}
	}

	// Set Defaults for comparison
	previousRelease.Release.SetDefaults(release.AwsRegion, release.AwsAccountID, "coinbase-odin-")
	previousRelease.SetDefaults()
	return release.validateSafeRelease(&previousRelease)
}

func (release *Release) validateSafeRelease(previousRelease *Release) error {
	// 1. Subnets, Image, or Services
	if res := safeUnorderedStrList(release.Subnets, previousRelease.Subnets); res != nil {
		return fmt.Errorf("SafeRelease Error: Subnets different %q", *res)
	}

	if res := safeStr(release.Image, previousRelease.Image); res != nil {
		return fmt.Errorf("SafeRelease Error: Image different %q", *res)
	}

	if res := safeInt(release.Timeout, previousRelease.Timeout); res != nil {
		return fmt.Errorf("SafeRelease Error: Timeout different %q", *res)
	}

	if err := validateSafeServices(release.Services, previousRelease.Services); err != nil {
		return err
	}

	return nil
}

// TODO better
func validateSafeServices(services map[string]*Service, prevServices map[string]*Service) error {

	if len(services) != len(prevServices) {
		// TODO better error message
		return fmt.Errorf("Services incorrect")
	}

	for serviceName, service := range services {
		prevService, ok := prevServices[serviceName]

		if !ok {
			return fmt.Errorf("SafeRelease Error(%v): No previous service", serviceName)
		}

		if err := validaeSafeService(serviceName, service, prevService); err != nil {
			return err
		}
	}

	return nil
}

func validaeSafeService(serviceName string, service *Service, prevService *Service) error {
	// 2. Security Groups or Profile

	if res := safeUnorderedStrList(service.SecurityGroups, prevService.SecurityGroups); res != nil {
		return fmt.Errorf("SafeRelease Error(%v): SecurityGroups different %q", serviceName, *res)
	}

	if res := safeStr(service.Profile, prevService.Profile); res != nil {
		return fmt.Errorf("SafeRelease Error(%v): Profile different %q", serviceName, *res)
	}

	// 3. ELBs or Target Groups
	if res := safeUnorderedStrList(service.ELBs, prevService.ELBs); res != nil {
		return fmt.Errorf("SafeRelease Error(%v): ELBs different %q", serviceName, *res)
	}

	if res := safeUnorderedStrList(service.TargetGroups, prevService.TargetGroups); res != nil {
		return fmt.Errorf("SafeRelease Error(%v): TargetGroups different %q", serviceName, *res)
	}

	// 5. EBS information
	if res := safeInt64(service.EBSVolumeSize, prevService.EBSVolumeSize); res != nil {
		return fmt.Errorf("SafeRelease Error(%v): EBSVolumeSize different %q", serviceName, *res)
	}

	if res := safeStr(service.EBSVolumeType, prevService.EBSVolumeType); res != nil {
		return fmt.Errorf("SafeRelease Error(%v): EBSVolumeType different %q", serviceName, *res)
	}

	if res := safeStr(service.EBSDeviceName, prevService.EBSDeviceName); res != nil {
		return fmt.Errorf("SafeRelease Error(%v): EBSDeviceName different %q", serviceName, *res)
	}

	// 6. AssociatePublicIpAddress
	if res := safeBool(service.AssociatePublicIpAddress, prevService.AssociatePublicIpAddress); res != nil {
		return fmt.Errorf("SafeRelease Error(%v): AssociatePublicIpAddress different %q", serviceName, *res)
	}

	// 4. Instance Type
	if res := safeStr(service.InstanceType, prevService.InstanceType); res != nil {
		return fmt.Errorf("SafeRelease Error(%v): InstanceType different %q", serviceName, *res)
	}

	if err := validaeSafeAutoscaling(serviceName, service.Autoscaling, prevService.Autoscaling); err != nil {
		return err
	}

	return nil
}

func validaeSafeAutoscaling(serviceName string, as *AutoScalingConfig, prevAs *AutoScalingConfig) error {
	if res := safeInt64(as.MinSize, prevAs.MinSize); res != nil {
		return fmt.Errorf("SafeRelease Error(%v): MinSize different %q", serviceName, *res)
	}

	if res := safeInt64(as.MaxSize, prevAs.MaxSize); res != nil {
		return fmt.Errorf("SafeRelease Error(%v): MaxSize different %q", serviceName, *res)
	}

	if res := safeInt64(as.MaxTerminations, prevAs.MaxTerminations); res != nil {
		return fmt.Errorf("SafeRelease Error(%v): MaxTerminations different %q", serviceName, *res)
	}

	if res := safeInt64(as.DefaultCooldown, prevAs.DefaultCooldown); res != nil {
		return fmt.Errorf("SafeRelease Error(%v): DefaultCooldown different %q", serviceName, *res)
	}

	if res := safeInt64(as.HealthCheckGracePeriod, prevAs.HealthCheckGracePeriod); res != nil {
		return fmt.Errorf("SafeRelease Error(%v): HealthCheckGracePeriod different %q", serviceName, *res)
	}

	if res := safeFloat64(as.Spread, prevAs.Spread); res != nil {
		return fmt.Errorf("SafeRelease Error(%v): Spread different %q", serviceName, *res)
	}

	// TODO: add Policy checks
	return nil
}

////
// Utils
////
func safeStr(s1 *string, s2 *string) *string {
	if s1 == nil && s2 == nil {
		return nil
	}

	if s1 == nil {
		return to.Strp(fmt.Sprintf("previous release has %v, requested nil", *s2))
	}

	if s2 == nil {
		return to.Strp(fmt.Sprintf("previous release has nil, requested %v", *s1))
	}

	if *s1 == *s2 {
		return nil
	}

	return to.Strp(fmt.Sprintf("previous release has %v, requested %v", *s2, *s1))
}

func safeInt64(s1 *int64, s2 *int64) *string {
	if s1 == nil && s2 == nil {
		return nil
	}

	if s1 == nil {
		return to.Strp(fmt.Sprintf("previous release has %v, requested nil", *s2))
	}

	if s2 == nil {
		return to.Strp(fmt.Sprintf("previous release has nil, requested %v", *s1))
	}

	if *s1 == *s2 {
		return nil
	}

	return to.Strp(fmt.Sprintf("previous release has %v, requested %v", *s2, *s1))
}

func safeInt(s1 *int, s2 *int) *string {
	if s1 == nil && s2 == nil {
		return nil
	}

	if s1 == nil {
		return to.Strp(fmt.Sprintf("previous release has %v, requested nil", *s2))
	}

	if s2 == nil {
		return to.Strp(fmt.Sprintf("previous release has nil, requested %v", *s1))
	}

	if *s1 == *s2 {
		return nil
	}

	return to.Strp(fmt.Sprintf("previous release has %v, requested %v", *s2, *s1))
}

func safeFloat64(s1 *float64, s2 *float64) *string {
	if s1 == nil && s2 == nil {
		return nil
	}

	if s1 == nil {
		return to.Strp(fmt.Sprintf("previous release has %v, requested nil", *s2))
	}

	if s2 == nil {
		return to.Strp(fmt.Sprintf("previous release has nil, requested %v", *s1))
	}

	if *s1 == *s2 {
		return nil
	}

	return to.Strp(fmt.Sprintf("previous release has %v, requested %v", *s2, *s1))
}

func safeBool(s1 *bool, s2 *bool) *string {
	if s1 == nil && s2 == nil {
		return nil
	}

	if s1 == nil {
		return to.Strp(fmt.Sprintf("previous release has %v, requested nil", *s2))
	}

	if s2 == nil {
		return to.Strp(fmt.Sprintf("previous release has nil, requested %v", *s1))
	}

	if *s1 == *s2 {
		return nil
	}

	return to.Strp(fmt.Sprintf("previous release has %v, requested %v", *s2, *s1))
}

func safeUnorderedStrList(s1 []*string, s2 []*string) *string {
	m1, ss1 := strS2Map(s1)
	m2, ss2 := strS2Map(s2)
	errStr := fmt.Sprintf(" previous release has %q, requested %q", ss2, ss1)
	if len(m1) != len(m2) {
		return &errStr
	}

	for s, _ := range m1 {
		_, ok := m2[s]
		if !ok {
			return &errStr
		}
	}

	return nil
}

func strS2Map(slc []*string) (map[string]bool, []string) {
	m := map[string]bool{}
	strSlice := []string{}
	for _, s := range slc {
		if s == nil {
			continue
		}
		m[*s] = true
		strSlice = append(strSlice, *s)
	}
	return m, strSlice
}
