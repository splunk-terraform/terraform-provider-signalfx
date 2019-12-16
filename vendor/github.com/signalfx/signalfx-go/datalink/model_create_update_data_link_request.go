package datalink

type CreateUpdateDataLinkRequest struct {
	// Name (key) of the metadata that's the trigger of a data link.
	PropertyName string `json:"propertyName,omitempty"`
	// Value of the metadata that's the trigger of a data link.
	PropertyValue string `json:"propertyValue,omitempty"`

	// Optional dashboard id
	ContextId string `json:"contextId,omitempty"`

	Targets []*Target `json:"targets,omitempty"`
}
