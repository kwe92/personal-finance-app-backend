package loader

import (
	"log"
	"os"

	"github.com/joho/godotenv"
	"personal_finance_backend/database"
	"personal_finance_backend/plaid_config"
)

func LoadEnv() {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, continuing with environment variables")
	}
}

func LoadPlaidConfig() {
	plaid_config.LoadPlaidConfig()
	database.InitializeStore()
}

func GetPort() string {
	if port := os.Getenv("APP_PORT"); port != "" {
		return port
	}
	return "8080"
}
