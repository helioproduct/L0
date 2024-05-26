package config

import "github.com/kelseyhightower/envconfig"

type DBConfig struct {
	DBname string `envconfig:"POSTGRES_DB" default:"servicedb"`
	User   string `envconfig:"POSTGRES_SERVICE_USER" default:"serviceuser"`
	Pass   string `envconfig:"POSTGRES_SERVICE_PASSWORD" default:"servicepassword"`
	Host   string `envconfig:"POSTGRES_HOST" default:"127.0.0.1"`
	Port   string `envconfig:"POSTGRES_PORT" default:"5432"`
}

type NATSConfig struct {
	ClusterID string `envconfig:"NATS_CLUSTER_ID" default:"test-cluster"`
	ClientID  string `envconfig:"NATS_CLIENT_ID" default:"test-client"`
	URL       string `envconfig:"NATS_URL" default:"nats://localhost:4222"`
}

type HTTPConfig struct {
	Addr string `envconfig:"HTTP_ADDR" default:"127.0.0.1:8080"`
}

type Config struct {
	DB   DBConfig
	NATS NATSConfig
	HTTP HTTPConfig
}

func GetConfig() (*Config, error) {
	var cfg Config
	err := envconfig.Process("", &cfg)

	return &cfg, err
}
