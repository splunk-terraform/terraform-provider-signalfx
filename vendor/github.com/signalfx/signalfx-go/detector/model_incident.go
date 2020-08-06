package detector

type Incident struct {
	Active       bool                    `json:"active,omitempty"`
	AnomalyState string                  `json:"anomalyState,omitempty"`
	DetectLabel  string                  `json:"detectLabel,omitempty"`
	DetectorId   string                  `json:"detectorId,omitempty"`
	DetectorName string                  `json:"detectorName,omitempty"`
	Events       []*Event                `json:"events,omitempty"`
	IncidentId   string                  `json:"incidentId,omitempty"`
	Inputs       *map[string]interface{} `json:"inputs,omitempty"`
	Severity     string                  `json:"severity,omitempty"`
}
