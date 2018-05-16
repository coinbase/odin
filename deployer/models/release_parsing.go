package models

import (
	"bytes"
	"encoding/json"
)

// The goal here is to raise an error if a key is sent that is not supported.
// This should stop many dangerous problems, like misspelling a parameter.
type _Release Release

// But the problem is that there are exceptions that we have
type _ReleaseExceptions struct {
	_Release
	Task *string // Do not include the Task because that can be implemented
}

// UnmarshalJSON should error if there is something unexpected
func (release *Release) UnmarshalJSON(data []byte) error {
	var releaseE _ReleaseExceptions
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields() // Force

	if err := dec.Decode(&releaseE); err != nil {
		return err
	}

	*release = Release(releaseE._Release)
	return nil
}
