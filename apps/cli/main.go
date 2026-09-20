package main

import (
	"os"
	"url-shortener-cli/internal/command"
)

var version = "dev"

func main() { os.Exit(command.Execute(command.Dependencies{Version: version}, os.Args[1:])) }
