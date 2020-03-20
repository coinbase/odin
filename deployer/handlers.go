package deployer

import (
	"context"
	"fmt"

	"github.com/coinbase/step/errors"
	"github.com/coinbase/step/utils/to"

	"github.com/coinbase/odin/aws"
	"github.com/coinbase/odin/deployer/models"
)

// DeployHandler function type
type DeployHandler func(context.Context, *models.Release) (*models.Release, error)

// Errors
type DetachError struct {
	Cause string
}

func (e DetachError) Error() string {
	return fmt.Sprintf("DetachError: %v", e.Cause)
}

////////////
// HANDLERS
////////////

var assumedRole = to.Strp("coinbase-odin-assumed")

// Validate checks the release for issues
func Validate(awsc aws.Clients) DeployHandler {
	return func(ctx context.Context, release *models.Release) (*models.Release, error) {
		// Assign the release its SHA before anything alters it
		release.ReleaseSHA256 = to.SHA256Struct(release)
		release.WipeControlledValues()

		// Default the releases Account and Region to where the Lambda is running
		region, account := to.AwsRegionAccountFromContext(ctx)
		release.Release.SetDefaults(region, account, "coinbase-odin-")
		release.SetDefaults() // Fill in all the blank Attributes

		if err := release.Validate(awsc.S3Client(release.AwsRegion, nil, nil)); err != nil {
			return nil, &errors.BadReleaseError{err.Error()}
		}

		return release, nil
	}
}

// Lock Tries to Grab the Lock, if it fails for any reason, no cleanup is necessary
func Lock(awsc aws.Clients) DeployHandler {
	return func(ctx context.Context, release *models.Release) (*models.Release, error) {
		release.SetDefaults()
		return release, release.GrabLocks(awsc.S3Client(release.AwsRegion, nil, nil))
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
			return nil, &errors.BadReleaseError{err.Error()}
		}

		if err := release.ValidateResources(resources); err != nil {
			return nil, &errors.BadReleaseError{err.Error()}
		}

		// If this flag is set Odin will fail a deploy if previous Release is dangerously different
		if release.SafeRelease {
			if err := release.ValidateSafeRelease(
				awsc.S3Client(release.AwsRegion, nil, nil),
				resources,
			); err != nil {
				return nil, &errors.BadReleaseError{err.Error()}
			}
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
		if err := release.SetDefaultsWithUserData(awsc.S3Client(release.AwsRegion, nil, nil)); err != nil {
			return nil, &errors.HaltError{err.Error()}
		}

		if err := release.IsHalt(awsc.S3Client(release.AwsRegion, nil, nil)); err != nil {
			return nil, &errors.HaltError{err.Error()}
		}

		if err := release.CreateResources(
			awsc.ASGClient(release.AwsRegion, release.AwsAccountID, assumedRole),
			awsc.CWClient(release.AwsRegion, release.AwsAccountID, assumedRole),
			awsc.ALBClient(release.AwsRegion, release.AwsAccountID, assumedRole),
		); err != nil {
			return nil, &errors.DeployError{err.Error()}
		}

		return release, nil
	}
}

// CheckHealthy checks all the instances are healthy
func CheckHealthy(awsc aws.Clients) DeployHandler {
	return func(_ context.Context, release *models.Release) (*models.Release, error) {
		release.SetDefaults() // Wire up non-serialized relationships

		if err := release.IsHalt(awsc.S3Client(release.AwsRegion, nil, nil)); err != nil {
			return nil, &errors.HaltError{err.Error()}
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
				return nil, &errors.HaltError{err.Error()}
			default:
				// This will retry a few times, as it might just be an AWS issue
				return nil, &errors.HealthError{err.Error()}
			}
		}

		return release, nil
	}
}

// DetachForSuccess detach ASGs
func DetachForSuccess(awsc aws.Clients) DeployHandler {
	return func(_ context.Context, release *models.Release) (*models.Release, error) {
		release.SetDefaults() // Wire up non-serialized relationships

		if err := release.DetachForSuccess(
			awsc.ASGClient(release.AwsRegion, release.AwsAccountID, assumedRole),
		); err != nil {
			switch err.(type) {
			case models.DetachError:
				return nil, &DetachError{err.Error()}
			default:
				return nil, &errors.CleanUpError{err.Error()}
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
			return nil, &errors.CleanUpError{err.Error()}
		}

		if err := release.UnlockRoot(awsc.S3Client(release.AwsRegion, nil, nil)); err != nil {
			return nil, &errors.LockError{err.Error()}
		}

		if err := release.ResetDesiredCapacity(
			awsc.ASGClient(release.AwsRegion, release.AwsAccountID, assumedRole),
		); err != nil {
			// We ignore this error as failing to reset the capacity should not cause a massive issue
			// Log the error in case
			fmt.Printf("IGNORED: %v \n", err)
		}

		release.RemoveHalt(awsc.S3Client(release.AwsRegion, nil, nil)) // Delete Halt

		release.Success = to.Boolp(true) // Wait till the end to mark success

		return release, nil
	}
}

// DetachForFailure detach ASGs
func DetachForFailure(awsc aws.Clients) DeployHandler {
	return func(_ context.Context, release *models.Release) (*models.Release, error) {
		release.SetDefaults() // Wire up non-serialized relationships

		if err := release.DetachForFailure(
			awsc.ASGClient(release.AwsRegion, release.AwsAccountID, assumedRole),
		); err != nil {
			switch err.(type) {
			case models.DetachError:
				return nil, &DetachError{err.Error()}
			default:
				return nil, &errors.CleanUpError{err.Error()}
			}
		}

		return release, nil
	}
}

// CleanUpFailure deletes newly deployed resources
func CleanUpFailure(awsc aws.Clients) DeployHandler {
	return func(_ context.Context, release *models.Release) (*models.Release, error) {
		release.SetDefaults() // Wire up non-serialized relationships

		release.Success = to.Boolp(false) // Quickly Mark Failure

		if err := release.UnsuccessfulTearDown(
			awsc.ASGClient(release.AwsRegion, release.AwsAccountID, assumedRole),
			awsc.CWClient(release.AwsRegion, release.AwsAccountID, assumedRole),
		); err != nil {
			switch err.(type) {
			case models.DetachError:
				return nil, &DetachError{err.Error()}
			default:
				return nil, &errors.CleanUpError{err.Error()}
			}
		}

		return release, nil
	}
}

// ReleaseLockFailure releases the lock then fails
func ReleaseLockFailure(awsc aws.Clients) DeployHandler {
	return func(_ context.Context, release *models.Release) (*models.Release, error) {
		release.SetDefaults() // Wire up non-serialized relationships

		if err := release.UnlockRoot(awsc.S3Client(release.AwsRegion, nil, nil)); err != nil {
			return nil, &errors.LockError{err.Error()}
		}

		release.RemoveHalt(awsc.S3Client(release.AwsRegion, nil, nil)) // Delete Halt

		return release, nil
	}
}
