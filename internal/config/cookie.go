package config

import "net/http"

func (c *Config) CookieSameSite() http.SameSite {
	switch c.CookieSameSiteMode {
	case "strict":
		return http.SameSiteStrictMode
	case "none":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteLaxMode
	}
}

