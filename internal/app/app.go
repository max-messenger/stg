package app

import "go.uber.org/fx"

func CreateApp(cfgName string) *fx.App {
	box := NewBox(
		Modules,
		WithAppName("stg"),
		WithConfigFile(cfgName),
	)

	return box.CreateApp()
}
