package config

import "os"

type Config struct {
	Name string
}

func Load() Config {
	name := os.Getenv("CONSUMER_NAME")
	if name == "" {
		name = "consumers"
	}
	return Config{Name: name}
}
