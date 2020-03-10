package dashboard

type DiscoveryOptions struct {
	Query     string    `json:"query,omitempty"`
	Selectors *[]string `json:"selectors,omitempty"`
}
