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

	// Image storage.
	StorageBackend   string `env:"BOOKOWL_STORAGE_BACKEND" envDefault:"local"`
	StorageLocalPath string `env:"BOOKOWL_STORAGE_LOCAL_PATH" envDefault:"/tmp/bookowl-images"`

	// S3-compatible storage.
	StorageS3Endpoint        string `env:"BOOKOWL_STORAGE_S3_ENDPOINT"`
	StorageS3Bucket          string `env:"BOOKOWL_STORAGE_S3_BUCKET" envDefault:"bookowl-images"`
	StorageS3Region          string `env:"BOOKOWL_STORAGE_S3_REGION" envDefault:"eu-central-1"`
	StorageS3AccessKeyID     string `env:"BOOKOWL_STORAGE_S3_ACCESS_KEY_ID"`
	StorageS3SecretAccessKey string `env:"BOOKOWL_STORAGE_S3_SECRET_ACCESS_KEY"`
	StorageS3PublicURL       string `env:"BOOKOWL_STORAGE_S3_PUBLIC_URL"`
	StorageS3UsePresign      bool   `env:"BOOKOWL_STORAGE_S3_USE_PRESIGN" envDefault:"false"`

	// Runbook migration.
	MigrateTenantSlug       string `env:"BOOKOWL_MIGRATE_TENANT_SLUG" envDefault:"acme"`
	MigrateTargetSpace      string `env:"BOOKOWL_MIGRATE_TARGET_SPACE" envDefault:"runbooks"`
	MigrateTargetCollection string `env:"BOOKOWL_MIGRATE_TARGET_COLLECTION" envDefault:"imported"`
}

func Load() (Config, error) {
	return env.ParseAs[Config]()
}
