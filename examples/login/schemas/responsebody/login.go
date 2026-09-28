package responsebody

// Login is the wire shape returned on successful authentication.
type Login struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"` // seconds
}

// Me is the wire shape returned by GET /me: the verified caller's user ID.
type Me struct {
	UserID string `json:"user_id"`
}
