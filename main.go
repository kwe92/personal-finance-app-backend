package main

import (
	"personal_finance_backend/database"
	"personal_finance_backend/loader"
	"personal_finance_backend/plaid_config"
	"personal_finance_backend/router"
)

func main() {
	loader.LoadEnv()
	loader.InitialConfigs()
	r := router.SetupRouter(plaid_config.Client, *database.DefaultStore)

	if err := r.Run(":" + loader.GetPort()); err != nil {
		panic("unable to start server: " + err.Error())
	}
}
