/*
Package main is used only during startup of the Application and will trigger the initial
Run function from the cmd package
*/
package main

import (
	"os"

	"github.com/P1llus/ess-openapi-servicebroker/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}

}
