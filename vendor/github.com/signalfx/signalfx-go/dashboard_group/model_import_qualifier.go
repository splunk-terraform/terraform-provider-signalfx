package dashboard_group

type ImportQualifier struct {
	Metric  string          `json:"metric,omitempty"`
	Filters []*ImportFilter `json:"filters,omitempty"`
}
