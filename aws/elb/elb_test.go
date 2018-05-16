package elb

import (
	"sort"
	"testing"

	"github.com/coinbase/odin/aws/mocks"
	"github.com/coinbase/step/utils/to"
	"github.com/stretchr/testify/assert"
)

func Test_GetInstances(t *testing.T) {
	//func GetInstances(elbc aws.ELBAPI, name *string, instances []string) (aws.Instances, error) {
	elbc := &mocks.ELBClient{}
	_, err := GetInstances(elbc, to.Strp("asd"), []string{"asd"})
	assert.Error(t, err)

	elbc.AddELB("asd", "project", "config", "service")
	ins, err := GetInstances(elbc, to.Strp("asd"), []string{"asd"})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(ins))
}

func Test_FindAll(t *testing.T) {
	//func FindAll(elbc aws.ELBAPI, names []*string) ([]*LoadBalancer, error) {
	elbc := &mocks.ELBClient{}
	_, err := FindAll(elbc, []*string{to.Strp("asd")})
	assert.Error(t, err)

	elbc.AddELB("asd", "project", "config", "service")
	elbc.AddELB("das", "project", "config", "service")

	elbs, err := FindAll(elbc, []*string{to.Strp("asd"), to.Strp("das")})

	assert.NoError(t, err)
	assert.Equal(t, 2, len(elbs))
}

func Test_createDescribeInstanceHealthInput(t *testing.T) {
	name := ""

	in := createDescribeInstanceHealthInput(&name, []string{})
	assert.Equal(t, len(in.Instances), 0)

	in = createDescribeInstanceHealthInput(&name, []string{"a"})
	assert.Equal(t, len(in.Instances), 1)
	assert.Equal(t, *in.Instances[0].InstanceId, "a")

	in = createDescribeInstanceHealthInput(&name, []string{"a", "b"})

	elbsIDs := []string{}
	for _, lb := range in.Instances {
		elbsIDs = append(elbsIDs, *lb.InstanceId)
	}

	sort.Strings(elbsIDs) // Sort not Guaranteed by map

	assert.Equal(t, len(elbsIDs), 2)
	assert.Equal(t, elbsIDs[0], "a")
	assert.Equal(t, elbsIDs[1], "b")
}
