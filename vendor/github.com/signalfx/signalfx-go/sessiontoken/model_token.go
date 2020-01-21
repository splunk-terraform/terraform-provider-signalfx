package sessiontoken

// Properties of a session token, in the form of a JSON object
type Token struct {
	// The user session token used for API requests
	AccessToken    string      `json:"accessToken"`
	// unknown (not documented)
	AuthMethod     string      `json:"authMethod"`
	// The internal user ID of the user who created the token
	CreatedBy      string	   `json:"createdBy"`
	// The date and time that the token was created, in Unix time This property is set by the system, and you can't change it.
	CreatedMs      int64         `json:"createdMs"`
	// Indicates if the token is disabled or not. When you first create a token, the value of disabled is false.
	Disabled       bool        `json:"disabled"`
	// The email address submitted in the request to create the token
	Email          string      `json:"email"`
	// The date and time that the token will expire, in Unix time
	ExpiryMs       int64       `json:"expiryMs"`
	// The SignalFx identifier of this access token
	ID             string      `json:"id"`
	// The SignalFx identifier of the organization that the user belongs to
	OrganizationID string      `json:"organizationId"`
	// unknown (not documented)
	PersonaID      string      `json:"personaId"`
	// unknown (not documented)
	ReadOnly       bool        `json:"readOnly"`
	// Always set to ORG_USER
	SessionType    string      `json:"sessionType"`
	// The date and time that the token was updated, in Unix time For a successful "create token" request, this value is the same as that for createdMs. This value is set by the system, and you can't change it.
	UpdatedMs      int         `json:"updatedMs"`
	// The SignalFx identifier of the user who created the token
	UserID         string      `json:"userId"`
}