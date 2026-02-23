package config

import "github.com/caarlos0/env/v11"

type Config struct {
	Mode    string `env:"BOOKOWL_MODE" envDefault:"api"`
	Host    string `env:"BOOKOWL_HOST" envDefault:"0.0.0.0"`
	Port    int    `env:"BOOKOWL_PORT" envDefault:"8081"`
	DevMode bool   `env:"BOOKOWL_DEV_MODE" envDefault:"false"`

	DBURL    string `env:"BOOKOWL_DB_URL" envDefault:"postgres://bookowl:bookowl@localhost:5433/bookowl?sslmode=disable"`
	RedisURL string `env:"BOOKOWL_REDIS_URL" envDefault:"redis://localhost:6380/0"`

	OIDCIssuer   string `env:"BOOKOWL_OIDC_ISSUER"`
	OIDCClientID string `env:"BOOKOWL_OIDC_CLIENT_ID" envDefault:"bookowl"`

	NightOwlAPIURL string `env:"BOOKOWL_NIGHTOWL_API_URL" envDefault:"http://localhost:8080"`
	NightOwlAPIKey string `env:"BOOKOWL_NIGHTOWL_API_KEY"`
}

func Load() (Config, error) {
	return env.ParseAs[Config]()
}
