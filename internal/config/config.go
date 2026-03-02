package config

import (
	"os"
	"strconv"
)

type Config struct {
	ServiceName   string
	HTTPAddr      string
	PostgresURL   string
	KafkaBrokers  []string
	RedisAddr     string
	MemcachedAddr string
	JWTPrivateKey string
	JWTPublicKey  string
	DemoMode      bool
}

func Load(service string) Config {
	return Config{
		ServiceName:   service,
		HTTPAddr:      getenv("HTTP_ADDR", ":8080"),
		PostgresURL:   getenv("POSTGRES_URL", "postgres://sentinel:sentinel@postgres:5432/sentinel?sslmode=disable"),
		KafkaBrokers:  []string{getenv("KAFKA_BROKER", "redpanda:9092")},
		RedisAddr:     getenv("REDIS_ADDR", "redis:6379"),
		MemcachedAddr: getenv("MEMCACHED_ADDR", "memcached:11211"),
		JWTPrivateKey: getenv("JWT_PRIVATE_KEY", "./keys/jwtRS256.key"),
		JWTPublicKey:  getenv("JWT_PUBLIC_KEY", "./keys/jwtRS256.key.pub"),
		DemoMode:      getbool("DEMO_MODE", true),
	}
}

func getenv(k, d string) string {
	v := os.Getenv(k)
	if v == "" {
		return d
	}
	return v
}
func getbool(k string, d bool) bool {
	v := os.Getenv(k)
	if v == "" {
		return d
	}
	b, _ := strconv.ParseBool(v)
	return b
}
