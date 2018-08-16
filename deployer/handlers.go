package deployer

import (
	"context"

	"github.com/coinbase/odin/aws"
	"github.com/coinbase/odin/deployer/models"
	"github.com/coinbase/step/errors"
	"github.com/coinbase/step/utils/to"
)

// DeployHandler function type
type DeployHandler func(context.Context, *models.Release) (*models.Release, error)

////////////
// HANDLERS
////////////

var assumedRole = to.Strp("coinbase-odin-assumed")

// Validate checks the release for issues
func Validate(awsc aws.Clients) DeployHandler {
	return func(ctx context.Context, release *models.Release) (*models.Release, error) {
		// Assign the release its SHA before anything alters it
		release.ReleaseSHA256 = to.SHA256Struct(release)

		// Default the releases Account and Region to where the Lambda is running
		region, account := to.AwsRegionAccountFromContext(ctx)
		release.Release.SetDefaults(region, account, "coinbase-odin-")
		release.SetDefaults() // Fill in all the blank Attributes

		if err := release.Validate(awsc.S3Client(nil, nil, nil)); err != nil {
			return nil, &errors.BadReleaseError{err.Error()}
		}

		return release, nil
	}
}

// Lock Tries to Grab the Lock, if it fails for any reason, no cleanup is necessary
func Lock(awsc aws.Clients) DeployHandler {
	return func(ctx context.Context, release *models.Release) (*models.Release, error) {
		release.SetDefaults()
		return release, release.GrabLock(awsc.S3Client(nil, nil, nil))
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
			return nil, &errors.BadReleaseError{err.Error()}
		}

		if err := release.IsHalt(awsc.S3Client(nil, nil, nil)); err != nil {
			return nil, &errors.HaltError{err.Error()}
		}

		if err := release.CreateResources(
			awsc.ASGClient(release.AwsRegion, release.AwsAccountID, assumedRole),
			awsc.CWClient(release.AwsRegion, release.AwsAccountID, assumedRole),
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

		if err := release.IsHalt(awsc.S3Client(nil, nil, nil)); err != nil {
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

		if err := release.ReleaseLock(awsc.S3Client(nil, nil, nil)); err != nil {
			return nil, &errors.LockError{err.Error()}
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
			return nil, &errors.CleanUpError{err.Error()}
		}

		return release, nil
	}
}

// ReleaseLockFailure releases the lock then fails
func ReleaseLockFailure(awsc aws.Clients) DeployHandler {
	return func(_ context.Context, release *models.Release) (*models.Release, error) {
		release.SetDefaults() // Wire up non-serialized relationships

		if err := release.ReleaseLock(awsc.S3Client(nil, nil, nil)); err != nil {
			return nil, &errors.LockError{err.Error()}
		}

		release.RemoveHalt(awsc.S3Client(nil, nil, nil)) // Delete Halt

		return release, nil
	}
}
