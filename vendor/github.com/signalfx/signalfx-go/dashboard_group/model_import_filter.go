package dashboard_group

// A single filter object to apply to the charts in the dashboard. The filter specifies a default or user-defined dimension or custom property. You can either include or exclude all the data that matches the dimension or custom property.
type ImportFilter struct {
	// Flag that indicates how the filter should operate. If `true`, data that matches the criteria is _excluded_ from charts; otherwise, data that matches the criteria is included.
	NOT bool `json:"NOT,omitempty"`
	// Name of the dimension or custom property to match to the data.<br> **Note:** If the dimension or custom property doesn't exist in any of the charts for the dashboard, and `ChartsFilterObject.NOT` is `true`, the system doesn't display any data in the charts.
	Property string `json:"property"`
	// A list of values to compare to the value of the dimension or custom property specified in `ChartsFilterObject.property`. If the list contains more than one value, the filter becomes a set of queries between the value of `property` and each element of `value`. The system joins these queries with an implicit OR.
	Values []string `json:"values,omitempty"`
}
