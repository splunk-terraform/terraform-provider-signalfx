package datalink

type CreateUpdateDataLinkRequest struct {
	// Name (key) of the metadata that's the trigger of a data link.
	PropertyName string `json:"propertyName,omitempty"`
	// Value of the metadata that's the trigger of a data link.
	PropertyValue string `json:"propertyValue,omitempty"`

	Targets []*Target `json:"targets,omitempty"`
}
