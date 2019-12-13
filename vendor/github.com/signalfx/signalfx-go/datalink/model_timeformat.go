package datalink

// TimeFormat : Designates the format of minimumTimeWindow in the same data link target object.
type TimeFormat string

// List of Time Format
const (
	ISO8601 TimeFormat = "ISO8601"
	Epoch   TimeFormat = "Epoch"
)
