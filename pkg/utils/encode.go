package utils

import (
	"encoding/json"
)

type CrashJson [][]string

// public
func (self *CrashJson) ToJson() ([]byte, error) {
	return json.Marshal(self)
}

// public
func (self *CrashJson) FromJson(data []byte) error {
	return json.Unmarshal(data, self)
}
