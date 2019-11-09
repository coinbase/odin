package models

import (
	"fmt"

	"github.com/coinbase/odin/aws"
	"github.com/coinbase/step/aws/s3"
	"github.com/coinbase/step/bifrost"
	"github.com/coinbase/step/utils/is"
	"github.com/coinbase/step/utils/to"
)

// Release is the Data Structure passed between Client to Deployer
type Release struct {
	bifrost.Release

	SafeRelease bool `json:"safe_release,omitempty"`

	Subnets []*string `json:"subnets,omitempty"`

	Image *string `json:"ami,omitempty"`

	userdata       *string // Not serialized
	UserDataSHA256 *string `json:"user_data_sha256,omitempty"`

	// LifeCycleHooks
	LifeCycleHooks map[string]*LifeCycleHook `json:"lifecycle,omitempty"`

	// Maintain a Log to look at what has happened
	Healthy *bool `json:"healthy,omitempty"`

	WaitForHealthy *int `json:"wait_for_healthy,omitempty"`
	WaitForDetach  *int `json:"wait_for_detach,omitempty"`

	// AWS Service is Downloaded
	Services map[string]*Service `json:"services,omitempty"` // Downloaded From S3
}

//////////
// Getters
//////////

// UserDataPath returns
func (release *Release) UserDataPath() *string {
	s := fmt.Sprintf("%v/userdata", *release.ReleaseDir())
	return &s
}

//////////
// Setters
//////////

// SetDefaultsWithUserData sets the default values including userdata fetched from S3
func (release *Release) SetDefaultsWithUserData(s3c aws.S3API) error {
	release.SetDefaults()
	err := release.DownloadUserData(s3c)
	if err != nil {
		return err
	}

	for _, service := range release.Services {
		if service != nil {
			service.SetUserData(release.UserData())
		}
	}

	return nil
}

// SetDefaults assigns default values
func (release *Release) SetDefaults() {
	// Overwrite WaitForHealthy to be Min 15 seconds, Max 5 minutes
	waitForHealthy := 120

	if release.Timeout == nil {
		release.Timeout = to.Intp(600)
	}

	switch {
	case *release.Timeout < 1800:
		// Under 30 mins check every 15 seconds
		waitForHealthy = 15
	case *release.Timeout < 7200:
		// Under 2 hour check every 60 seconds
		waitForHealthy = 60
	}

	release.WaitForHealthy = to.Intp(waitForHealthy)

	// Default to 20 if WaitForDetach
	if release.WaitForDetach == nil || *release.WaitForDetach < 5 {
		release.WaitForDetach = to.Intp(10)
	}

	if release.Healthy == nil {
		release.Healthy = to.Boolp(false)
	}

	if release.LifeCycleHooks == nil {
		release.LifeCycleHooks = map[string]*LifeCycleHook{}
	}

	for name, lc := range release.LifeCycleHooks {
		if lc != nil {
			lc.SetDefaults(release.AwsRegion, release.AwsAccountID, name)
		}
	}

	for name, service := range release.Services {
		if service != nil {
			service.SetDefaults(release, name)
		}
	}
}

//////////
// Validate
//////////

// Validate returns
func (release *Release) Validate(s3c aws.S3API) error {
	if err := release.Release.Validate(s3c, &Release{}); err != nil {
		return err
	}

	// Max timeout is 48 hours (for now)
	if *release.Timeout > 172800 {
		// 48 hours of timeout means the WaitForHealthy of 120 will work
		return fmt.Errorf("%v Max timeout is 172800 (48 hours)", release.ErrorPrefix())
	}

	if (5.0/float64(*release.WaitForHealthy))*(float64(*release.Timeout)) > 10000.0 {
		// There are 5 state transitions per health check
		// (5/WaitForHealthy) * Timeout is about equal to the max state transistions
		// Due to limitations on StepFucntions History Events the max state transistions is about 10k
		// So (5/WaitForHealthy) * Timeout < 10k as a rule of thumb
		return fmt.Errorf("%v Rule of Thumb (5/WaitForHealthy) * Timeout < 10k", release.ErrorPrefix())
	}

	if release.Image == nil {
		return fmt.Errorf("%v %v", release.ErrorPrefix(), "AMI image must be provided")
	}

	if err := release.ValidateUserDataSHA(s3c); err != nil {
		return fmt.Errorf("%v %v", release.ErrorPrefix(), err.Error())
	}

	if err := release.ValidateServices(); err != nil {
		return fmt.Errorf("%v %v", release.ErrorPrefix(), err.Error())
	}

	return nil
}

// ValidateUserDataSHA validates the userdata has the correct SHA for the release
func (release *Release) ValidateUserDataSHA(s3c aws.S3API) error {
	if is.EmptyStr(release.UserDataSHA256) {
		return fmt.Errorf("UserDataSHA256 must be defined")
	}

	err := release.DownloadUserData(s3c)

	if err != nil {
		return fmt.Errorf("Error Getting UserData with %v", err.Error())
	}

	userdataSha := to.SHA256Str(release.UserData())
	if userdataSha != *release.UserDataSHA256 {
		return fmt.Errorf("UserData SHA incorrect expected %v, got %v", userdataSha, *release.UserDataSHA256)
	}

	return nil
}

// UserData returns user data
func (release *Release) UserData() *string {
	return release.userdata
}

// DownloadUserData fetches and populates the User data from S3
func (release *Release) DownloadUserData(s3c aws.S3API) error {
	userdataBytes, err := s3.Get(s3c, release.Bucket, release.UserDataPath())

	if err != nil {
		return err
	}

	release.SetUserData(to.Strp(string(*userdataBytes)))
	return nil
}

// SetUserData sets the User data
func (release *Release) SetUserData(userdata *string) {
	release.userdata = userdata
}

// ValidateServices returns
func (release *Release) ValidateServices() error {
	if release.Services == nil {
		return fmt.Errorf("Services nil")
	}

	if len(release.Services) == 0 {
		return fmt.Errorf("Services empty")
	}

	for name, service := range release.Services {
		if service == nil {
			return fmt.Errorf("Service %v is nil", name)
		}

		err := service.Validate()
		if err != nil {
			return err
		}
	}

	return nil
}
