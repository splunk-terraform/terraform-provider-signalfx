package integration

// Specifies the data collection integration between Microsoft Azure and SignalFx, in the form of a JSON object.
type JiraIntegration struct {
	// The creation date and time for the integration object, in Unix time UTC-relative. The system sets this value, and you can't modify it.
	Created int64 `json:"created,omitempty"`
	// SignalFx-assigned user ID of the user that created the integration object. If the system created the object, the value is \"AAAAAAAAAA\". The system sets this value, and you can't modify it.
	Creator string `json:"creator,omitempty"`
	// Flag that indicates the state of the integration object. If  `true`, the integration is enabled. If `false`, the integration is disabled, and you must enable it by setting \"enabled\" to `true` in a **PUT** request that updates the object. <br> **NOTE:** SignalFx always sets the flag to `true` when you call  **POST** `/integration` to create an integration.
	Enabled bool `json:"enabled"`
	// SignalFx-assigned ID of an integration you create in the web UI or API. Use this property to retrieve an integration using the **GET**, **PUT**, or **DELETE** `/integration/{id}` endpoints or the **GET** `/integration/validate{id}/` endpoint, as described in this topic.
	Id string `json:"id,omitempty"`
	// The last time the integration was updated, in Unix time UTC-relative. This value is \"read-only\".
	LastUpdated int64 `json:"lastUpdated,omitempty"`
	// SignalFx-assigned ID of the last user who updated the integration. If the last update was by the system, the value is \"AAAAAAAAAA\". This value is \"read-only\".
	LastUpdatedBy string `json:"lastUpdatedBy,omitempty"`
	// A human-readable label for the integration. This property helps you identify a specific integration when you're using multiple integrations for the same service.
	Name string `json:"name,omitempty"`
	Type Type   `json:"type"`

	APIToken   string        `json:"apiToken,omitempty"`
	UserEmail  string        `json:"userEmail,omitempty"`
	Username   string        `json:"username,omitempty"`
	Password   string        `json:"password,omitempty"`
	Assignee   *JiraAssignee `json:"assignee,omitempty"`
	AuthMethod string        `json:"authMethod"`
	BaseURL    string        `json:"baseUrl"`
	IssueType  string        `json:"issueType"`
	ProjectKey string        `json:"projectKey"`
}
