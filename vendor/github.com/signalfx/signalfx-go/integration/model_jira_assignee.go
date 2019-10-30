package integration

type JiraAssignee struct {
	// Jira display name for the assignee
	DisplayName string `json:"displayName,omitempty"`
	// Jira user name for the assignee
	Name string `json:"name"`
}
