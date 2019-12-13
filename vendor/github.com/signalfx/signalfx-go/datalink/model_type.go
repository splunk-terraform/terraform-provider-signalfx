package datalink

// Type : Target type designator
type Type string

// List of Type
const (
	INTERNAL_LINK Type = "INTERNAL_LINK"
	EXTERNAL_LINK Type = "EXTERNAL_LINK"
	SPLUNK_LINK   Type = "SPLUNK_LINK"
)
