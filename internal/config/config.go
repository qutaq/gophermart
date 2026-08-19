package config

import (
	"crypto/rand"
	"encoding/hex"
	"flag"
	"os"
)

type Config struct {
	RunAddress           string
	DatabaseURI          string
	AccrualSystemAddress string
	JWTSecret            string
}

func Parse() Config {
	cfg := Config{}

	flag.StringVar(&cfg.RunAddress, "a", "localhost:8080",
		"адрес и порт запуска сервиса (env RUN_ADDRESS)")
	flag.StringVar(&cfg.DatabaseURI, "d", "",
		"DSN подключения к PostgreSQL (env DATABASE_URI)")
	flag.StringVar(&cfg.AccrualSystemAddress, "r", "",
		"адрес системы расчёта начислений (env ACCRUAL_SYSTEM_ADDRESS)")
	flag.Parse()

	if v, ok := os.LookupEnv("RUN_ADDRESS"); ok {
		cfg.RunAddress = v
	}
	if v, ok := os.LookupEnv("DATABASE_URI"); ok {
		cfg.DatabaseURI = v
	}
	if v, ok := os.LookupEnv("ACCRUAL_SYSTEM_ADDRESS"); ok {
		cfg.AccrualSystemAddress = v
	}
	if v, ok := os.LookupEnv("JWT_SECRET"); ok {
		cfg.JWTSecret = v
	}
	if cfg.JWTSecret == "" {
		b := make([]byte, 32)
		if _, err := rand.Read(b); err != nil {
			panic("config: не удалось сгенерировать JWT_SECRET: " + err.Error())
		}
		cfg.JWTSecret = hex.EncodeToString(b)
	}

	return cfg
}
