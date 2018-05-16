package aws

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_MergeInstances(t *testing.T) {
	i1 := Instances{"i": healthy}
	i2 := Instances{"i": healthy}
	assert.Equal(t, healthy, i1.MergeInstances(i2)["i"])

	i2 = Instances{"i": unhealthy}
	assert.Equal(t, unhealthy, i1.MergeInstances(i2)["i"])

	i2 = Instances{"i": terminating}
	assert.Equal(t, terminating, i1.MergeInstances(i2)["i"])

	// inverse
	i2 = Instances{"i": healthy}
	assert.Equal(t, healthy, i2.MergeInstances(i1)["i"])

	i2 = Instances{"i": unhealthy}
	assert.Equal(t, unhealthy, i2.MergeInstances(i1)["i"])

	i2 = Instances{"i": terminating}
	assert.Equal(t, terminating, i2.MergeInstances(i1)["i"])
}
