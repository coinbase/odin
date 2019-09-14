package models

import (
	"fmt"

	"github.com/coinbase/step/aws"
	"github.com/coinbase/step/aws/s3"
	"github.com/coinbase/step/bifrost"
)

//////////
// Safe Deploy
//////////

// ValidateSafeDeploy will error if the currently deployed release has different:
// 1. Subnets, Image, or Services
// Or any service has different:
// 2. Security Groups or Profile
// 3. ELBs or Target Groups
// 4. Instance Type or Autoscaling Preferences
// 5. EBS information
// 6. AssociatePublicIpAddress
func (release *Release) ValidateSafeDeploy(s3c aws.S3API, resources *ReleaseResources) error {
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
		return err
	}

	return release.validateSafeDeploy(&previousRelease)
}

func (release *Release) validateSafeDeploy(previousRelease *Release) error {
	// 1. Subnets, Image, or Services
	if !equalUnorderedStrList(release.Subnets, previousRelease.Subnets) {
		return fmt.Errorf("SafeDeploy Error: Subnets different")
	}

	if !equalStr(release.Image, previousRelease.Image) {
		return fmt.Errorf("SafeDeploy Error: Image different")
	}

	if err := safeServices(release.Services, previousRelease.Services); err != nil {
		return err
	}

	return nil
}

// TODO better
func safeServices(services map[string]*Service, prevServices map[string]*Service) error {

	if len(services) != len(prevServices) {
		// TODO better error message
		return fmt.Errorf("Services incorrect")
	}

	for serviceName, service := range services {
		prevService, ok := prevServices[serviceName]
		if !ok {
			return false
		}

		if err := safeService(service, prevService); err != nil {
			return err
		}
	}

	return true
}

func safeService(service *Service, prevService *Service) error {
	// 2. Security Groups or Profile
	// 3. ELBs or Target Groups
	// 4. Instance Type or Autoscaling Preferences
	// 5. EBS information
	// 6. AssociatePublicIpAddress
	if !equalUnorderedStrList(service.SecurityGroups, prevService.SecurityGroups) {
		return fmt.Errorf("SafeDeploy Error: SecurityGroups different")
	}

	if !equalUnorderedStrList(service.ELBs, prevService.ELBs) {
		return fmt.Errorf("SafeDeploy Error: ELBs different")
	}

	if !equalUnorderedStrList(service.TargetGroups, prevService.TargetGroups) {
		return fmt.Errorf("SafeDeploy Error: ELBs different")
	}

	return nil
}

////
// Utils
////
func equalStr(s1 *string, s2 *string) bool {
	if s1 == nil {
		return false
	}

	if s2 == nil {
		return false
	}

	return *s1 == *s2
}

func equalUnorderedStrList(s1 []*string, s2 []*string) bool {
	m1 := strS2Map(s1)
	m2 := strS2Map(s2)
	if len(m1) != len(m2) {
		return false
	}

	for s, _ := range m1 {
		_, ok := m2[s]
		if !ok {
			return false
		}
	}

	return true
}

func strS2Map(slc []*string) map[string]bool {
	m := map[string]bool{}
	for _, s := range slc {
		if s == nil {
			continue
		}
		m[*s] = true
	}
	return m
}
