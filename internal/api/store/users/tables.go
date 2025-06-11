package users

type ORCIDAuthorization struct {
	Name         string `json:"name"`
	ORCID        string `json:"orcid"`
	Scope        string `json:"scope"`
	ExpiresIn    int64  `json:"expires_in"`
	TokenType    string `json:"token_type"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}
