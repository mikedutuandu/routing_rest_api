package main

// set default UTC time zone
import _ "janio-backend/tzinit"

import (
	"janio-backend/config"
	"janio-backend/cron"
	"janio-backend/db"
	"janio-backend/route"
)

func main() {
	db.Init()
	go cron.Init1()
	e := route.Init()

	config := config.GetConfig()

	e.Server.Addr = ":" + config.RUN_PORT

	// Serve it like a boss
	//e.Logger.Fatal(gracehttp.Serve(e.Server))
	e.Logger.Fatal(e.Start(":" + config.RUN_PORT))
}
