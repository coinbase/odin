package sg

import (
	"testing"

	"github.com/coinbase/odin/aws/mocks"
	"github.com/coinbase/step/utils/to"
	"github.com/stretchr/testify/assert"
)

func Test_Find(t *testing.T) {
	//func Find(ec2Client aws.EC2API, name_tags []*string) ([]*SecurityGroup, error) {
	ec2c := &mocks.EC2Client{}
	_, err := Find(ec2c, []*string{to.Strp("sg1")})
	assert.Error(t, err)

	ec2c.AddSecurityGroup("sg1", "project_name", "config_name", "service_name", nil)

	sgs, err := Find(ec2c, []*string{to.Strp("sg1")})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(sgs))
}
