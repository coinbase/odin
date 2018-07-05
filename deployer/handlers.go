package deployer

import (
	"context"
	"fmt"

	"github.com/coinbase/odin/aws"
	"github.com/coinbase/odin/deployer/models"
	"github.com/coinbase/step/utils/to"
)

// DeployHandler function type
type DeployHandler func(context.Context, *models.Release) (*models.Release, error)

////////////
// ERRORS
////////////

// ErrorWrapper error
type ErrorWrapper struct {
	err error
}

func (e *ErrorWrapper) Error() string {
	return fmt.Sprintf("ERROR: %v", e.err)
}

// BadReleaseError error
type BadReleaseError struct {
	*ErrorWrapper
}

// LockExistsError error
type LockExistsError struct {
	*ErrorWrapper
}

// LockError error
type LockError struct {
	*ErrorWrapper
}

// DeployError error
type DeployError struct {
	*ErrorWrapper
}

// HealthError error
type HealthError struct {
	*ErrorWrapper
}

// HaltError error
type HaltError struct {
	*ErrorWrapper
}

// CleanUpError error
type CleanUpError struct {
	*ErrorWrapper
}

func throw(err error) error {
	fmt.Printf("%v: %v\n", to.ErrorType(err), err.Error())
	return err
}

////////////
// HANDLERS
////////////

var assumedRole = to.Strp("coinbase-odin-assumed")

// Validate checks the release for issues
func Validate(awsc aws.Clients) DeployHandler {
	return func(ctx context.Context, release *models.Release) (*models.Release, error) {
		// Assign the release its SHA before anything alters it
		release.SetReleaseSHA256(to.SHA256Struct(release))

		// Default the releases Account and Region to where the Lambda is running
		release.SetDefaultRegionAccount(to.AwsRegionAccountFromContext(ctx))
		release.SetUUID()     // Ensure that this is set by Server
		release.SetDefaults() // Fill in all the blank Attributes

		if err := release.Validate(awsc.S3Client(nil, nil, nil)); err != nil {
			return nil, throw(&BadReleaseError{&ErrorWrapper{err}})
		}

		return release, nil
	}
}

// Lock Tries to Grab the Lock, if it fails for any reason, no cleanup is necessary
func Lock(awsc aws.Clients) DeployHandler {
	return func(ctx context.Context, release *models.Release) (*models.Release, error) {
		release.SetDefaults() // Wire up non-serialized relationships

		// First Thing is to grab the Lock
		grabbed, err := release.GrabLock(awsc.S3Client(nil, nil, nil))

		// Check grabbed first because there are errors that can be thrown before anything is created
		if !grabbed {
			if err != nil {
				return nil, throw(&LockExistsError{&ErrorWrapper{err}})
			}

			return nil, throw(&LockExistsError{&ErrorWrapper{fmt.Errorf("Lock Already Exists")}})
		}

		if err != nil {
			return nil, throw(&LockError{&ErrorWrapper{err}})
		}

		return release, nil
	}
}

// ValidateResources ensures resources exist, and are valid
// It also retrieves and saves necessary information about those resources
func ValidateResources(awsc aws.Clients) DeployHandler {
	return func(_ context.Context, release *models.Release) (*models.Release, error) {
		release.SetDefaults() // Wire up non-serialized relationships

		// Fetch all Resource Objecgs from AWS, i.e. Security Group, ELBs, Albs, IAM Profile
		resources, err := release.FetchResources(
			awsc.ASGClient(release.AwsRegion, release.AwsAccountID, assumedRole),
			awsc.EC2Client(release.AwsRegion, release.AwsAccountID, assumedRole),
			awsc.ELBClient(release.AwsRegion, release.AwsAccountID, assumedRole),
			awsc.ALBClient(release.AwsRegion, release.AwsAccountID, assumedRole),
			awsc.IAMClient(release.AwsRegion, release.AwsAccountID, assumedRole),
			awsc.SNSClient(release.AwsRegion, release.AwsAccountID, assumedRole),
		)

		if err != nil {
			return nil, throw(&BadReleaseError{&ErrorWrapper{err}})
		}

		if err := release.ValidateResources(resources); err != nil {
			return nil, throw(&BadReleaseError{&ErrorWrapper{err}})
		}

		release.UpdateWithResources(resources)

		return release, nil
	}
}

// Deploy receives release, fetches AWS cloud resources, and creates New resources
// It returns the release with additional information including
func Deploy(awsc aws.Clients) DeployHandler {
	return func(_ context.Context, release *models.Release) (*models.Release, error) {
		// Wire up non-serialized relationships with UserData
		if err := release.SetDefaultsWithUserData(awsc.S3Client(nil, nil, nil)); err != nil {
			return nil, throw(&BadReleaseError{&ErrorWrapper{err}})
		}

		if err := release.IsHalt(awsc.S3Client(nil, nil, nil)); err != nil {
			return nil, throw(&HaltError{&ErrorWrapper{err}})
		}

		if err := release.CreateResources(
			awsc.ASGClient(release.AwsRegion, release.AwsAccountID, assumedRole),
			awsc.CWClient(release.AwsRegion, release.AwsAccountID, assumedRole),
		); err != nil {
			return nil, throw(&DeployError{&ErrorWrapper{err}})
		}

		return release, nil
	}
}

// CheckHealthy checks all the instances are healthy
func CheckHealthy(awsc aws.Clients) DeployHandler {
	return func(_ context.Context, release *models.Release) (*models.Release, error) {
		release.SetDefaults() // Wire up non-serialized relationships

		if err := release.IsHalt(awsc.S3Client(nil, nil, nil)); err != nil {
			return nil, throw(&HaltError{&ErrorWrapper{err}})
		}

		err := release.UpdateHealthy(
			awsc.ASGClient(release.AwsRegion, release.AwsAccountID, assumedRole),
			awsc.ELBClient(release.AwsRegion, release.AwsAccountID, assumedRole),
			awsc.ALBClient(release.AwsRegion, release.AwsAccountID, assumedRole),
		)

		if err != nil {
			switch err.(type) {
			case *models.HaltError:
				// This will immediately stop checking and fail the deploy
				return nil, throw(&HaltError{&ErrorWrapper{err}})
			default:
				// This will retry a few times, as it might just be an AWS issue
				return nil, throw(&HealthError{&ErrorWrapper{err}})
			}
		}

		return release, nil
	}
}

// CleanUpSuccess deleted the old resources
func CleanUpSuccess(awsc aws.Clients) DeployHandler {
	return func(_ context.Context, release *models.Release) (*models.Release, error) {
		release.SetDefaults() // Wire up non-serialized relationships

		if err := release.SuccessfulTearDown(
			awsc.ASGClient(release.AwsRegion, release.AwsAccountID, assumedRole),
			awsc.CWClient(release.AwsRegion, release.AwsAccountID, assumedRole),
		); err != nil {
			return nil, throw(&CleanUpError{&ErrorWrapper{err}})
		}

		if err := release.ReleaseLock(awsc.S3Client(nil, nil, nil)); err != nil {
			return nil, throw(&LockError{&ErrorWrapper{err}})
		}

		release.RemoveHalt(awsc.S3Client(nil, nil, nil)) // Delete Halt

		release.Success = to.Boolp(true) // Wait till the end to mark success

		return release, nil
	}
}

// CleanUpFailure deletes newly deployed resources
func CleanUpFailure(awsc aws.Clients) DeployHandler {
	return func(_ context.Context, release *models.Release) (*models.Release, error) {
		release.SetDefaults() // Wire up non-serialized relationships

		release.Success = to.Boolp(false) // Quickly Mark Failure

		if err := release.UnsuccssfulTearDown(
			awsc.ASGClient(release.AwsRegion, release.AwsAccountID, assumedRole),
			awsc.CWClient(release.AwsRegion, release.AwsAccountID, assumedRole),
		); err != nil {
			return nil, throw(&CleanUpError{&ErrorWrapper{err}})
		}

		return release, nil
	}
}

// ReleaseLockFailure releases the lock then fails
func ReleaseLockFailure(awsc aws.Clients) DeployHandler {
	return func(_ context.Context, release *models.Release) (*models.Release, error) {
		release.SetDefaults() // Wire up non-serialized relationships

		if err := release.ReleaseLock(awsc.S3Client(nil, nil, nil)); err != nil {
			return nil, throw(&LockError{&ErrorWrapper{err}})
		}

		release.RemoveHalt(awsc.S3Client(nil, nil, nil)) // Delete Halt

		return release, nil
	}
}
