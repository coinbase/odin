package models

import (
	"fmt"
	"testing"

	"github.com/coinbase/odin/aws/mocks"
	"github.com/coinbase/step/utils/to"
	"github.com/stretchr/testify/assert"
)

func Test_Service_SetGetUserdata(t *testing.T) {
	release := MockMinimalRelease(t)

	service := Service{}
	service.SetUserData(to.Strp("{{RELEASE_ID}}\n{{PROJECT_NAME}}\n{{CONFIG_NAME}}\n{{SERVICE_NAME}}\n"))
	service.SetDefaults(release, "web")

	assert.Equal(t, fmt.Sprintf("%v\n%v\n%v\nweb\n", *release.ReleaseID, *release.ProjectName, *release.ConfigName), *service.UserData())
}

func Test_Service_CreateInput_HealthCheckGracePeriod(t *testing.T) {
	release := MockMinimalRelease(t)
	release.Timeout = to.Intp(10)

	service := Service{}
	service.SetUserData(to.Strp("{{RELEASE_ID}}\n{{PROJECT_NAME}}\n{{CONFIG_NAME}}\n{{SERVICE_NAME}}\n"))
	service.SetDefaults(release, "web")

	input := service.createInput()
	assert.Equal(t, *input.HealthCheckGracePeriod, int64(10))
}

func Test_Service_PlacementgroupValidation(t *testing.T) {
	// bad strat
	service := Service{
		PlacementGroupName:     to.Strp("asd"),
		PlacementGroupStrategy: to.Strp("asd"),
	}

	err := service.validatePlacementGroupAttributes()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "PlacementGroupStrategy")

	// need PartitionCount
	service = Service{
		PlacementGroupName:     to.Strp("asd"),
		PlacementGroupStrategy: to.Strp("partition"),
	}

	err = service.validatePlacementGroupAttributes()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "PlacementGroupPartitionCount")

	// need only if partitionPartitionCount
	service = Service{
		PlacementGroupName:           to.Strp("asd"),
		PlacementGroupStrategy:       to.Strp("spread"),
		PlacementGroupPartitionCount: to.Int64p(10),
	}

	err = service.validatePlacementGroupAttributes()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "PlacementGroupPartitionCount")

	// need PartitionCount
	service = Service{
		PlacementGroupName:           to.Strp("asd"),
		PlacementGroupStrategy:       to.Strp("partition"),
		PlacementGroupPartitionCount: to.Int64p(10),
	}

	err = service.validatePlacementGroupAttributes()
	assert.NoError(t, err)

}

func Test_Service_SetDesiredCapacity_Works(t *testing.T) {
	service := Service{
		Autoscaling: &AutoScalingConfig{
			MinSize: to.Int64p(int64(4)),
			MaxSize: to.Int64p(int64(10)),
			Spread:  to.Float64p(float64(0.8)),
		},
		PreviousDesiredCapacity: to.Int64p(6),
	}

	awsc := mocks.MockAWS()

	assert.NoError(t, service.SetDesiredCapacity(awsc.ASG))
	assert.Equal(t, int64(6), *awsc.ASG.SetDesiredCapacityLastInput.DesiredCapacity)
}

func Test_Service_CapacityValues(t *testing.T) {
	service := Service{
		Autoscaling: &AutoScalingConfig{
			MinSize: to.Int64p(int64(10)),
			MaxSize: to.Int64p(int64(50)),
			Spread:  to.Float64p(float64(0.8)),
		},
		PreviousDesiredCapacity: to.Int64p(20),
	}

	assert.Equal(t, 10, service.target())          // The number of instances we want healthy
	assert.Equal(t, 20, service.desiredCapacity()) // The final number of instances
	assert.Equal(t, 36, service.targetCapacity())  // The number of launched instances
}
