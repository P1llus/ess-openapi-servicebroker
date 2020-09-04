package main

import (
	"fmt"
	"os"

	"github.com/P1llus/ess-openapi-servicebroker/cmd"
)

func main() {

	if err := cmd.Run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

}
