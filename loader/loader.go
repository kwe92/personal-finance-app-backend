package loader

import (
	"log"
	"os"

	"personal_finance_backend/database"
	"personal_finance_backend/plaid_config"

	"github.com/joho/godotenv"
)

func LoadEnv() {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, continuing with environment variables")
	}
}

func InitialConfigs() {
	plaid_config.LoadPlaidConfig()
	database.InitializeStore()
}

func GetPort() string {
	if port := os.Getenv("APP_PORT"); port != "" {
		return port
	}
	return "8080"
}
