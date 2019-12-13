package datalink

import "github.com/signalfx/signalfx-go/util"

type Target struct {
	Type Type `json:"type,omitempty"`
	// SignalFx-assigned ID of the dashboard link target's dashboard group
	DashboardGroupId string `json:"dashboardGroupId,omitempty"`
	// User-assigned name of a link target dashboard's dashboard group.
	DashboardGroupName string `json:"dashboardGroupName,omitempty"`
	// SignalFx-assigned ID of the dashboard link target
	DashboardId string `json:"dashboardId,omitempty"`
	// User-assigned name of the dashboard link target.
	DashboardName string `json:"dashboardName,omitempty"`
	// Flag that designates a target as the default for a data link object.
	IsDefault bool `json:"isDefault,omitempty"`
	// User-assigned target name.
	Name string `json:"name,omitempty"`
	// The minimum time window for a search sent to an external site.
	MinimumTimeWindow util.StringOrInteger `json:"minimumTimeWindow,omitempty"`
	// Describes the relationship between SignalFx metadata keys and external system properties when the key names are different
	PropertyKeyMapping map[string]string `json:"propertyKeyMapping,omitempty"`
	TimeFormat         TimeFormat        `json:"timeFormat,omitempty"`
	URL                string            `json:"url,omitempty"`
}
