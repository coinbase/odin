package subnet

import (
	"testing"

	"github.com/coinbase/odin/aws/mocks"
	"github.com/coinbase/step/utils/to"
	"github.com/stretchr/testify/assert"
)

func Test_Find_IDs_and_Tags(t *testing.T) {
	//func Find(ec2Client aws.EC2API, name_tags_or_ids []*string) ([]*Subnet, error) {
	ec2c := &mocks.EC2Client{}
	_, err := Find(ec2c, []*string{to.Strp("private-subnet")})
	assert.Error(t, err)

	ec2c.AddSubnet("private-subnet1", "subnet-asd1")

	sgs, err := Find(ec2c, []*string{to.Strp("private-subnet1")})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(sgs))

	sgs, err = Find(ec2c, []*string{to.Strp("subnet-asd1")})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(sgs))
}

func Test_isID(t *testing.T) {
	assert.True(t, isID("subnet-asfasf"))
	assert.False(t, isID("ubuntu"))
	assert.False(t, isID("subnetgfjosd"))
}

func Test_splitIDsTags(t *testing.T) {
	ids := []*string{to.Strp("subnet-asfasf"), to.Strp("subnet-aasdasdf")}
	tags := []*string{to.Strp("privatea"), to.Strp("privateb")}

	unclean := []*string{to.Strp("privatea"), to.Strp("subnet-aasdasdf")}

	i, ts := splitIDsTags(ids)
	assert.Equal(t, 2, len(i))
	assert.Equal(t, 0, len(ts))

	i, ts = splitIDsTags(tags)
	assert.Equal(t, 0, len(i))
	assert.Equal(t, 2, len(ts))

	i, ts = splitIDsTags(unclean)
	assert.Equal(t, 1, len(i))
	assert.Equal(t, 1, len(ts))
}
