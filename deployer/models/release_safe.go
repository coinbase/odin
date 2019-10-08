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
// 1. Subnets, or Services
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

	// return an error for valid services
	return release.validateSafeRelease(&previousRelease)
}

type SafeReleaseError struct {
	Subnets        error
	Timeout        error
	AllServices    error
	MissingService error

	Services map[string]*SafeReleaseServiceError
}

type SafeReleaseServiceError struct {
	SecurityGroups           error
	Profile                  error
	ELBs                     error
	TargetGroups             error
	EBSVolumeSize            error
	EBSVolumeType            error
	EBSDeviceName            error
	AssociatePublicIpAddress error
	InstanceType             error
	MinSize                  error
	MaxSize                  error
	MaxTerminations          error
	DefaultCooldown          error
	HealthCheckGracePeriod   error
	Spread                   error
}

// Prints the list of safe release errors
func (sre *SafeReleaseError) Error() string {
	errstr := ""
	errstr = appendError(errstr, sre.Subnets)
	errstr = appendError(errstr, sre.Timeout)
	errstr = appendError(errstr, sre.AllServices)
	errstr = appendError(errstr, sre.MissingService)
	for _, srse := range sre.Services {
		errstr = appendError(errstr, srse.SecurityGroups)
		errstr = appendError(errstr, srse.Profile)
		errstr = appendError(errstr, srse.ELBs)
		errstr = appendError(errstr, srse.TargetGroups)
		errstr = appendError(errstr, srse.EBSVolumeSize)
		errstr = appendError(errstr, srse.EBSVolumeType)
		errstr = appendError(errstr, srse.EBSDeviceName)
		errstr = appendError(errstr, srse.AssociatePublicIpAddress)
		errstr = appendError(errstr, srse.InstanceType)
		errstr = appendError(errstr, srse.MinSize)
		errstr = appendError(errstr, srse.MaxSize)
		errstr = appendError(errstr, srse.MaxTerminations)
		errstr = appendError(errstr, srse.DefaultCooldown)
		errstr = appendError(errstr, srse.HealthCheckGracePeriod)
		errstr = appendError(errstr, srse.Spread)
	}

	return errstr
}

func appendError(errstr string, err error) string {
	if err != nil {
		errstr = fmt.Sprintf("%s\n%s", errstr, err.Error())
	}

	return errstr
}

func (release *Release) validateSafeRelease(previousRelease *Release) error {
	sre := &SafeReleaseError{
		Services: map[string]*SafeReleaseServiceError{},
	}

	// 1. Subnets, or Services
	if res := safeUnorderedStrList(release.Subnets, previousRelease.Subnets); res != nil {
		sre.Subnets = fmt.Errorf("SafeRelease Error: Subnets different %v", *res)
	}

	if res := safeInt(release.Timeout, previousRelease.Timeout); res != nil {
		sre.Timeout = fmt.Errorf("SafeRelease Error: Timeout different %v", *res)
	}

	// This will add errors to the sre
	validateSafeServices(sre, release.Services, previousRelease.Services)

	// Check whether an error was found and return if it has
	if sre.Error() == "" {
		return nil
	}

	return sre
}

func validateSafeServices(sre *SafeReleaseError, services map[string]*Service, prevServices map[string]*Service) {
	if res := safeUnorderedStrList(serviceMapKeys(services), serviceMapKeys(prevServices)); res != nil {
		sre.AllServices = fmt.Errorf("SafeRelease Error: Incorrect Services service %v", *res)
	}

	for serviceName, service := range services {
		prevService, ok := prevServices[serviceName]

		if !ok {
			// Pretty sure this will never be reached (best check though)
			sre.MissingService = fmt.Errorf("SafeRelease Error(%v): No previous service", serviceName)
		}

		sre.Services[serviceName] = validateSafeService(serviceName, service, prevService)
	}
}

func validateSafeService(serviceName string, service *Service, prevService *Service) *SafeReleaseServiceError {
	srse := &SafeReleaseServiceError{}
	// 2. Security Groups or Profile

	if res := safeUnorderedStrList(service.SecurityGroups, prevService.SecurityGroups); res != nil {
		srse.SecurityGroups = fmt.Errorf("SafeRelease Error(%v): SecurityGroups different %v", serviceName, *res)
	}

	if res := safeStr(service.Profile, prevService.Profile); res != nil {
		srse.Profile = fmt.Errorf("SafeRelease Error(%v): Profile different %v", serviceName, *res)
	}

	// 3. ELBs or Target Groups
	if res := safeUnorderedStrList(service.ELBs, prevService.ELBs); res != nil {
		srse.ELBs = fmt.Errorf("SafeRelease Error(%v): ELBs different %v", serviceName, *res)
	}

	if res := safeUnorderedStrList(service.TargetGroups, prevService.TargetGroups); res != nil {
		srse.TargetGroups = fmt.Errorf("SafeRelease Error(%v): TargetGroups different %v", serviceName, *res)
	}

	// 5. EBS information
	if res := safeInt64(service.EBSVolumeSize, prevService.EBSVolumeSize); res != nil {
		srse.EBSVolumeSize = fmt.Errorf("SafeRelease Error(%v): EBSVolumeSize different %v", serviceName, *res)
	}

	if res := safeStr(service.EBSVolumeType, prevService.EBSVolumeType); res != nil {
		srse.EBSVolumeType = fmt.Errorf("SafeRelease Error(%v): EBSVolumeType different %v", serviceName, *res)
	}

	if res := safeStr(service.EBSDeviceName, prevService.EBSDeviceName); res != nil {
		srse.EBSDeviceName = fmt.Errorf("SafeRelease Error(%v): EBSDeviceName different %v", serviceName, *res)
	}

	// 6. AssociatePublicIpAddress
	if res := safeBool(service.AssociatePublicIpAddress, prevService.AssociatePublicIpAddress); res != nil {
		srse.AssociatePublicIpAddress = fmt.Errorf("SafeRelease Error(%v): AssociatePublicIpAddress different %v", serviceName, *res)
	}

	// 4. Instance Type
	if res := safeStr(service.InstanceType, prevService.InstanceType); res != nil {
		srse.InstanceType = fmt.Errorf("SafeRelease Error(%v): InstanceType different %v", serviceName, *res)
	}

	validateSafeAutoscaling(srse, serviceName, service.Autoscaling, prevService.Autoscaling)

	return srse
}

func validateSafeAutoscaling(srse *SafeReleaseServiceError, serviceName string, as *AutoScalingConfig, prevAs *AutoScalingConfig) {
	if res := safeInt64(as.MinSize, prevAs.MinSize); res != nil {
		srse.MinSize = fmt.Errorf("SafeRelease Error(%v): MinSize different %v", serviceName, *res)
	}

	if res := safeInt64(as.MaxSize, prevAs.MaxSize); res != nil {
		srse.MaxSize = fmt.Errorf("SafeRelease Error(%v): MaxSize different %v", serviceName, *res)
	}

	if res := safeInt64(as.MaxTerminations, prevAs.MaxTerminations); res != nil {
		srse.MaxTerminations = fmt.Errorf("SafeRelease Error(%v): MaxTerminations different %v", serviceName, *res)
	}

	if res := safeInt64(as.DefaultCooldown, prevAs.DefaultCooldown); res != nil {
		srse.DefaultCooldown = fmt.Errorf("SafeRelease Error(%v): DefaultCooldown different %v", serviceName, *res)
	}

	if res := safeInt64(as.HealthCheckGracePeriod, prevAs.HealthCheckGracePeriod); res != nil {
		srse.HealthCheckGracePeriod = fmt.Errorf("SafeRelease Error(%v): HealthCheckGracePeriod different %v", serviceName, *res)
	}

	if res := safeFloat64(as.Spread, prevAs.Spread); res != nil {
		srse.Spread = fmt.Errorf("SafeRelease Error(%v): Spread different %v", serviceName, *res)
	}
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
	errStr := fmt.Sprintf("previous release has %v, requested %v", ss2, ss1)
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

func serviceMapKeys(sm map[string]*Service) []*string {
	strSlice := []*string{}
	for serviceName, _ := range sm {
		// Maintain ref
		a := serviceName
		strSlice = append(strSlice, &a)
	}
	return strSlice
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
