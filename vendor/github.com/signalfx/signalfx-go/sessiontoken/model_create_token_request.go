package sessiontoken

// Properties of a session token request.
type CreateTokenRequest struct {
	// The email address you used to join the organization for which you want a session token. Only used for Create Token requests
	Email string `json:"email"`
	// The password you provided to SignalFx when you accepted an invitation to join an organization. Only used for Create Token requests
	Password string `json:"password"`
	// Org to use - needed when the user uses same email in multiple orgs
	OrganizationId string `json:"organizationId"`
}
