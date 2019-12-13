package datalink

type DataLink struct {
	// The data link creation date and time, in the form of a Unix time value (milliseconds since the Unix epoch 1970-01-01 00:00:00 UTC+0). The system sets this value, and you can't modify it.
	Created int64 `json:"created,omitempty"`
	// SignalFx-assigned user ID of the user that created the data link. If the system created this dashboard, the value is \"AAAAAAAAAA\". The system sets this value, and you can't modify it.
	Creator string `json:"creator,omitempty"`
	// The data link's SignalFx-assigned ID. This value is \"read-only\" for a create request. The system assigns it and returns it to you in the response.
	Id string `json:"id,omitempty"`
	// The last time the data link was updated, in the form of a Unix timestamp (milliseconds since the Unix epoch 1970-01-01 00:00:00 UTC+0) This value is \"read-only\".
	LastUpdated int64 `json:"lastUpdated,omitempty"`
	// SignalFx-assigned ID of the last user who updated the data link. If the last update was by the system, the value is \"AAAAAAAAAA\". This value is \"read-only\".
	LastUpdatedBy string `json:"lastUpdatedBy,omitempty"`
	// Name (key) of the metadata that's the trigger of a data link.
	PropertyName string `json:"propertyName,omitempty"`
	// Value of the metadata that's the trigger of a data link.
	PropertyValue string `json:"propertyValue,omitempty"`

	Targets []*Target `json:"targets,omitempty"`
}
