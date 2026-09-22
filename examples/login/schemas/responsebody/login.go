package responsebody

// Login is the wire shape returned on successful authentication.
type Login struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"` // seconds
}
