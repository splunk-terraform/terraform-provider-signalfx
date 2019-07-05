package dashboard_group

// DashboardConfig is configurations associated with the dashboard group.
type DashboardConfig struct {
	ConfigId string `json:"configId,omitempty"`
	// DashboardId is SignalFx-assigned identifier for a dashboard. In a dashboard group, dashboard IDs track the dashboards associated with the group. If you try to update the ID of an existing configuration entry or use a non-existent ID, the system returns an error.
	DashboardId string `json:"dashboardId,omitempty"`
	// String that provides a description override for a mirrored dashboard
	DescriptionOverride string `json:"descriptionOverride,omitempty"`
	// Filter and dashboard variable overrides for the mirrored dashboard
	FiltersOverride Filters `json:"filters,omitempty"`
	// String that overrides the name of the dashboard specified in dashboardId. This property is primarily intended to provide a unique name for a mirrored dashboard.
	NameOverride string `json:"nameOverride,omitempty"`
}
