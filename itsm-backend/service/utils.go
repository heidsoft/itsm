package service

import (
	"encoding/json"
)

// SafeMarshal marshals an interface to JSON, returning a byte slice or empty JSON object on error
func SafeMarshal(v interface{}) []byte {
	if v == nil {
		return []byte("{}")
	}
	bytes, err := json.Marshal(v)
	if err != nil {
		return []byte("{}")
	}
	return bytes
}
