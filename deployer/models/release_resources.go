package models

import (
	"fmt"

	"github.com/coinbase/odin/aws"
	"github.com/coinbase/odin/aws/ami"
	"github.com/coinbase/odin/aws/asg"
	"github.com/coinbase/odin/aws/subnet"
)

//////////
// Validate Resources
//////////

// FetchResources checks the existence of all Resources references in this release
// and returns a struct of the resources
func (release *Release) FetchResources(asgc aws.ASGAPI, ec2 aws.EC2API, elbc aws.ELBAPI, albc aws.ALBAPI, iamc aws.IAMAPI, snsc aws.SNSAPI, pricec aws.PricingAPI) (map[string]*ServiceResources, error) {
	resources := map[string]*ServiceResources{}

	// If there are any ASGs with this release ID error
	badASGs, err := asg.ForProjectConfigReleaseID(asgc, release.ProjectName, release.ConfigName, release.ReleaseID)
	if err != nil {
		return nil, err
	}

	if len(badASGs) != 0 {
		return nil, fmt.Errorf("%v ASGs exist for same project config release", release.ErrorPrefix())
	}

	prevASGs, err := asg.ForProjectConfigNotReleaseIDServiceMap(asgc, release.ProjectName, release.ConfigName, release.ReleaseID)
	if err != nil {
		return nil, err
	}

	// Fetch Subnets
	subnets, err := subnet.Find(ec2, release.Subnets)
	if err != nil {
		return nil, err
	}

	// Fetch Image
	im, err := ami.Find(ec2, release.Image)
	if err != nil {
		return nil, err
	}

	// LifeCycleHooks
	for _, lc := range release.LifeCycleHooks {
		if err := lc.FetchResources(iamc, snsc); err != nil {
			return nil, err
		}
	}

	for name, service := range release.Services {
		sr, err := service.FetchResources(ec2, elbc, albc, iamc, pricec)
		if err != nil {
			return nil, err
		}

		sr.Subnets = subnets
		sr.Image = im
		sr.PrevASG = prevASGs[name]

		resources[name] = sr
	}

	return resources, nil
}

// ValidateResources returns
func (release *Release) ValidateResources(resources map[string]*ServiceResources) error {
	// Fetch Service
	for name, service := range release.Services {
		sr := resources[name]
		if sr == nil {
			return fmt.Errorf("%v ServiceResources nil for %v", release.ErrorPrefix(), name)
		}
		if err := sr.Validate(service); err != nil {
			return err
		}
	}
	return nil
}

// UpdateWithResources returns
func (release *Release) UpdateWithResources(resources map[string]*ServiceResources) {
	// Assign PreDesiredCapacity
	// Assign ServiceResourceName

	for name, service := range release.Services {
		sr := resources[name]
		if sr == nil {
			continue // Skip
		}
		if sr.PrevASG != nil {
			service.PreviousDesiredCapacity = sr.PrevASG.DesiredCapacity
		}

		service.Resources = sr.ToServiceResourceNames()
	}
}

//////////
// Create Resources
//////////

// CreateResources returns
func (release *Release) CreateResources(asgc aws.ASGAPI, cwc aws.CWAPI) error {
	for _, service := range release.Services {
		err := service.CreateResources(asgc, cwc)
		if err != nil {
			return err
		}
	}
	return nil
}

//////////
// Healthy Resources
//////////

// UpdateHealthy will try set the Healthy attribute
// First Error is a Halting Error, Second Error is a Retry Error
func (release *Release) UpdateHealthy(asgc aws.ASGAPI, elbc aws.ELBAPI, albc aws.ALBAPI) error {
	healthy := true

	for _, service := range release.Services {

		if err := service.UpdateHealthy(asgc, elbc, albc); err != nil {
			return err
		}

		healthy = healthy && service.Healthy // Healthy if all services are healthy
	}

	release.Healthy = &healthy

	return nil
}

//////////
// Teardown
//////////

// SuccessfulTearDown returns
func (release *Release) SuccessfulTearDown(asgc aws.ASGAPI, cwc aws.CWAPI) error {
	// Tear down all resources in NOT in this release
	asgs, err := asg.ForProjectConfigNOTReleaseID(asgc, release.ProjectName, release.ConfigName, release.ReleaseID)

	if err != nil {
		return err
	}

	// Delete all Previous Resources
	for _, asg := range asgs {
		if *release.ProjectName != *asg.ProjectName() {
			return fmt.Errorf("Bad Project")
		}

		if *release.ConfigName != *asg.ConfigName() {
			return fmt.Errorf("Bad Config")
		}

		if *release.ReleaseID == *asg.ReleaseID() {
			return fmt.Errorf("Bad ReleaseID")
		}

		if err := asg.Teardown(asgc, cwc); err != nil {
			return err
		}

	}

	return nil
}

// UnsuccessfulTearDown deletes the services we were trying to create because :(
func (release *Release) UnsuccessfulTearDown(asgc aws.ASGAPI, cwc aws.CWAPI) error {
	// Tear down all resources in this release
	asgs, err := asg.ForProjectConfigReleaseID(asgc, release.ProjectName, release.ConfigName, release.ReleaseID)
	if err != nil {
		return err
	}

	// Delete all Resources for this release
	for _, asg := range asgs {
		if *release.ProjectName != *asg.ProjectName() {
			return fmt.Errorf("Bad Project")
		}

		if *release.ConfigName != *asg.ConfigName() {
			return fmt.Errorf("Bad Config")
		}

		if *release.ReleaseID != *asg.ReleaseID() {
			return fmt.Errorf("Bad ReleaseID")
		}

		if asg.ReleaseID() == nil || *release.ReleaseID != *asg.ReleaseID() {
			return fmt.Errorf("Bad ReleaseID")
		}

		if err := asg.Teardown(asgc, cwc); err != nil {
			return err
		}
	}

	return nil
}
