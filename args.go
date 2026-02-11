package main

import (
	"flag"
	"log"
	"path/filepath"
)

type CliArguments struct {
	ConfigFile   string
	ListCommands bool
}

func InitArguments() CliArguments {
	configFilePath := flag.String("config", "config.yaml", "Path to `config.yaml` file")
	listCmds := flag.Bool("list", false, "List the commands available")

	flag.Parse()

	cfgPath, err := filepath.Abs(*configFilePath)
	if err != nil {
		log.Fatalf("Failed to parse config file path: %v", err)

	}

	args := CliArguments{
		ConfigFile:   cfgPath,
		ListCommands: *listCmds,
	}

	return args
}
