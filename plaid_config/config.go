package plaid_config

import (
	"log"
	"os"

	"github.com/plaid/plaid-go/v12/plaid"
)

var Client *plaid.APIClient
var PLAID_CLIENT_ID string
var PLAID_SECRET string
var PLAID_ENV string
var PLAID_PRODUCTS string
var PLAID_REDIRECT_URI string
var APP_PORT string

func LoadPlaidConfig() {
	PLAID_CLIENT_ID = os.Getenv("PLAID_CLIENT_ID")
	PLAID_SECRET = os.Getenv("PLAID_SECRET")
	PLAID_ENV = os.Getenv("PLAID_ENV")
	PLAID_PRODUCTS = os.Getenv("PLAID_PRODUCTS")
	PLAID_REDIRECT_URI = os.Getenv("PLAID_REDIRECT_URI")
	APP_PORT = os.Getenv("APP_PORT")

	if PLAID_CLIENT_ID == "" || PLAID_SECRET == "" || PLAID_ENV == "" {
		log.Println("plaid configuration is incomplete")
		return
	}

	configuration := plaid.NewConfiguration()
	configuration.AddDefaultHeader("PLAID-CLIENT-ID", PLAID_CLIENT_ID)
	configuration.AddDefaultHeader("PLAID-SECRET", PLAID_SECRET)
	configuration.UseEnvironment(plaid.Sandbox)

	switch PLAID_ENV {
	case "sandbox":
		configuration.UseEnvironment(plaid.Sandbox)
	case "development":
		configuration.UseEnvironment(plaid.Development)
	case "production":
		configuration.UseEnvironment(plaid.Production)
	default:
		configuration.UseEnvironment(plaid.Sandbox)
	}

	Client = plaid.NewAPIClient(configuration)
}
