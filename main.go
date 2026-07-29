package main

import (
	"personal_finance_backend/loader"
	"personal_finance_backend/router"
)

func main() {
	loader.LoadEnv()
	loader.LoadPlaidConfig()

	r := router.SetupRouter()

	if err := r.Run(":" + loader.GetPort()); err != nil {
		panic("unable to start server: " + err.Error())
	}
}
