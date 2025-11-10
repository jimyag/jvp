package config

type Config struct{}

func New() (*Config, error) {
	return &Config{}, nil
}
