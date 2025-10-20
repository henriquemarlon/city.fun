package main

import (
	"os"

	"github.com/henriquemarlon/city.fun/simulator/cmd/congo/root"
)

func main() {
	err := root.Cmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
