package pg

import (
	"fmt"

	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/coinbase/odin/aws"
	"github.com/coinbase/step/utils/to"
)

// FindOrCreatePartitionGroup will find a partition group by name.
// If it doenst exist:
// - It will create one with the provided detail (error if detail is nil)
// If one does exist:
// - It will fetch that Partition Group and Error if any provided detail is missing
func FindOrCreatePartitionGroup(ec2c aws.EC2API, groupName *string, partitionCount *int64, strategy *string) error {
	if groupName == nil {
		return fmt.Errorf("PlacementGroupError: groupName nil")
	}

	pg, err := findPlacementGroup(ec2c, groupName)
	if err != nil {
		return err
	}

	// If no pg is found create a new one
	// otherwise validate that the found one equals the correct values
	if pg == nil {
		return createNewPlacementGroup(ec2c, groupName, partitionCount, strategy)
	}

	return validatePlacementGroup(pg, groupName, partitionCount, strategy)
}

// findPlacementGroup will search for a placement group, if none are found it will return (nil, nil)
func findPlacementGroup(ec2c aws.EC2API, groupName *string) (*ec2.PlacementGroup, error) {
	out, err := ec2c.DescribePlacementGroups(&ec2.DescribePlacementGroupsInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{
				Name:   to.Strp("group-name"),
				Values: []*string{groupName},
			},
		},
	})

	if err != nil {
		return nil, err
	}

	for _, checkPG := range out.PlacementGroups {
		if to.Strs(checkPG.GroupName) == to.Strs(groupName) {
			return checkPG, nil
		}
	}

	return nil, nil
}

func createNewPlacementGroup(ec2c aws.EC2API, groupName *string, partitionCount *int64, strategy *string) error {
	_, err := ec2c.CreatePlacementGroup(&ec2.CreatePlacementGroupInput{
		GroupName:      groupName,
		PartitionCount: partitionCount,
		Strategy:       strategy,
	})
	return err
}

func validatePlacementGroup(pg *ec2.PlacementGroup, groupName *string, partitionCount *int64, strategy *string) error {
	// pending | available | deleting | deleted
	if to.Strs(pg.State) != "available" {
		return fmt.Errorf("PlacementGroupError(%s): PG in invalid state %s", to.Strs(groupName), to.Strs(pg.State))
	}

	if to.Strs(groupName) != to.Strs(pg.GroupName) {
		return fmt.Errorf("PlacementGroupError(%s): PG in invalid name %s", to.Strs(groupName), to.Strs(pg.GroupName))
	}

	if to.Strs(strategy) != to.Strs(pg.Strategy) {
		return fmt.Errorf("PlacementGroupError(%s): PG in invalid strategy expected %s, got %s", to.Strs(groupName), to.Strs(pg.Strategy), to.Strs(strategy))
	}

	// Partition count should be equal only if strategy is 'partition'
	if *strategy != "partition" {
		return nil
	}

	if partitionCount == nil || pg.PartitionCount == nil {
		// We should never get here, but checking is easy
		return fmt.Errorf("PlacementGroupError(%s): PG has nil PartitionCount", to.Strs(groupName))
	}

	if *partitionCount != *pg.PartitionCount {
		return fmt.Errorf("PlacementGroupError(%s): PG in invalid strategy expected %q, got %q", to.Strs(groupName), *pg.PartitionCount, *partitionCount)
	}

	return nil
}
