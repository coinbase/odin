package cost

import (
	"testing"

	"github.com/coinbase/odin/aws/mocks"
	"github.com/coinbase/step/utils/to"
	"github.com/stretchr/testify/assert"
)

func Test_Cost(t *testing.T) {
	pricec := &mocks.PricingClient{}
	price, err := Cost(pricec, to.Strp("us-east-1"), to.Strp("r4.large"))

	assert.NoError(t, err)
	assert.Equal(t, *price, "0.100000")
}

func Test_SmartBidPrice(t *testing.T) {
	pricec := &mocks.PricingClient{}
	price, err := SmartBidPrice(pricec, to.Strp("us-east-1"), to.Strp("r4.large"))

	assert.NoError(t, err)
	assert.Equal(t, *price, "0.101000")
}
