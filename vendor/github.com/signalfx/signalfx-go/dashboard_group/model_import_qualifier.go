package dashboard_group

type ImportQualifier struct {
	Metric  string    `json:"metric,omitempty"`
	Filters []*Filter `json:"filters,omitempty"`
}
