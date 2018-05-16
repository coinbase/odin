package iam

import (
	"testing"

	"github.com/coinbase/odin/aws/mocks"
	"github.com/coinbase/step/utils/to"
	"github.com/stretchr/testify/assert"
)

func Test_Find(t *testing.T) {
	//func FindAll(elbc aws.ELBAPI, names []*string) ([]*LoadBalancer, error) {
	iamc := &mocks.IAMClient{}
	_, err := Find(iamc, to.Strp("asd"))
	assert.Error(t, err)

	iamc.AddGetInstanceProfile("asd", "/path/")
	profile, err := Find(iamc, to.Strp("asd"))
	assert.NoError(t, err)
	assert.Equal(t, "/path/", *profile.Path)
}
