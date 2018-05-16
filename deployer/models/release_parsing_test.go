package models

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Parsing_Errors_WithUnknownKey(t *testing.T) {
	var r Release
	assert.NoError(t, json.Unmarshal([]byte(`{}`), &r))

	assert.NoError(t, json.Unmarshal([]byte(`{"release_id" : "1"}`), &r))

	assert.Error(t, json.Unmarshal([]byte(`{"release_ids" : "1"}`), &r))
}
