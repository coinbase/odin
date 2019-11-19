package pg

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/coinbase/odin/aws/mocks"
	"github.com/coinbase/step/utils/to"
)

func Test_FindOrCreatePartitionGroup(t *testing.T) {
	//func Find(ec2Client aws.EC2API, name_tags_or_ids []*string) ([]*Subnet, error) {
	ec2c := &mocks.EC2Client{}

	// This will creates a placement group
	err := FindOrCreatePartitionGroup(ec2c, "i/i", to.Strp("i/i/groupName"), to.Int64p(10), to.Strp("cluster"))
	assert.NoError(t, err)

	// Finds the already created group
	err = FindOrCreatePartitionGroup(ec2c, "i/i", to.Strp("i/i/groupName"), to.Int64p(10), to.Strp("cluster"))
	assert.NoError(t, err)

	// This will error because the strategy is incorrect
	err = FindOrCreatePartitionGroup(ec2c, "i/i", to.Strp("i/i/groupName"), to.Int64p(10), to.Strp("wrong_stratgy"))
	assert.Error(t, err)

	err = FindOrCreatePartitionGroup(ec2c, "bad_prefix", to.Strp("i/i/groupName"), to.Int64p(10), to.Strp("wrong_stratgy"))
	assert.Error(t, err)
}
