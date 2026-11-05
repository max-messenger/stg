package main

import (
	"flag"

	"stg/internal/app"
)

func main() {
	var (
		cfgName string
	)

	flag.StringVar(&cfgName, "c", "config.yaml", "config")
	flag.Parse()

	app.CreateApp(cfgName).Run()
}
