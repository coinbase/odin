package ami

import (
	"testing"

	"github.com/coinbase/odin/aws/mocks"
	"github.com/coinbase/step/utils/to"
	"github.com/stretchr/testify/assert"
)

func Test_isID(t *testing.T) {
	assert.True(t, isID("ami-asfasf"))
	assert.False(t, isID("ubuntu"))
	assert.False(t, isID("amigdshiunet"))
	assert.False(t, isID("ami"))
}

func Test_Find_ID(t *testing.T) {
	ec2c := &mocks.EC2Client{}
	ec2c.AddImage("ubuntu", "ami-000000")
	img, err := Find(ec2c, to.Strp("ami-000000"))
	assert.NoError(t, err)
	assert.Equal(t, "ami-000000", *img.ImageID)
}

func Test_Find_Tag(t *testing.T) {
	ec2c := &mocks.EC2Client{}
	ec2c.AddImage("ubuntu", "ami-000000")
	img, err := Find(ec2c, to.Strp("ubuntu"))
	assert.NoError(t, err)
	assert.Equal(t, "ami-000000", *img.ImageID)
}
