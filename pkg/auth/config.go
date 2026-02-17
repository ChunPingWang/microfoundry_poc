package auth

// AuthConfig holds OIDC/Keycloak authentication settings.
type AuthConfig struct {
	Enabled      bool   `mapstructure:"enabled"`
	IssuerURL    string `mapstructure:"issuer_url"`
	ClientID     string `mapstructure:"client_id"`
	ClientSecret string `mapstructure:"client_secret"`
	RedirectURL  string `mapstructure:"redirect_url"`
	SessionKey   string `mapstructure:"session_key"`
}
