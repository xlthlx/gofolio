package model

// Setting is one persisted site setting, stored as a key/value pair.
type Setting struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}
