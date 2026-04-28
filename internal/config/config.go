package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	PORT       string
	DBUSER     string
	DBPASSWORD string
	DBHOST     string
	DBPORT     string
	DBNAME     string
}

func LoadConfig() Config {
	err := godotenv.Load()
	if err != nil {
		println("Error loading .env file")
	}

	cfg := Config{
		PORT:       os.Getenv("PORT"),
		DBUSER:     os.Getenv("DB_USER"),
		DBPASSWORD: os.Getenv("DB_PASSWORD"),
		DBHOST:     os.Getenv("DB_HOST"),
		DBPORT:     os.Getenv("DB_PORT"),
		DBNAME:     os.Getenv("DB_NAME"),
	}

	return cfg
}
