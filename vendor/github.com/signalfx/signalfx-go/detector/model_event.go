package detector

type Event struct {
	AnomalyState     string                  `json:"anomalyState,omitempty"`
	DetectLabel      string                  `json:"detectLabel,omitempty"`
	DetectorId       string                  `json:"detectorId,omitempty"`
	DetectorName     string                  `json:"detectorName,omitempty"`
	EventAnnotations *map[string]interface{} `json:"event_annotations,omitempty"`
	Id               string                  `json:"id,omitempty"`
	IncidentId       string                  `json:"incidentId,omitempty"`
	Inputs           *map[string]interface{} `json:"inputs,omitempty"`
	Severity         string                  `json:"severity,omitempty"`
	Timestamp        int64                   `json:"timestamp,omitempty"`
}
