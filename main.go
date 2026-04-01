package main

import "github.com/clazic/kosis-cli/cmd"

var version = "dev"

func main() {
	cmd.SetVersion(version)
	cmd.Execute()
}
