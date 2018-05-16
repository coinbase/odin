package models

import (
	"testing"

	"github.com/coinbase/step/utils/to"
	"github.com/stretchr/testify/assert"
)

func Test_Policy_Valid(t *testing.T) {
	pol := &Policy{
		Type: to.Strp("cpu_scale_down"),
	}

	pol.SetDefaults(to.Strp("service_id"))

	assert.NoError(t, pol.ValidateAttributes())
}
