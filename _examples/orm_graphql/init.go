package main

import (
	"flag"
	"fmt"
)

// Variables of program information
var (
	BuildVersion = "0.0.1"
	BuildName    = "example_orms"
	BuiltTime    = "2020-07-16 15:12:40"
	CommitID     = ""
	ConfigFile   = "../etc/config.yaml"
	MQConfigFile = "../etc/mq.yaml"
)

func init() {
	var showVer bool
	var confFile string

	flag.BoolVar(&showVer, "v", false, "Build version")
	flag.StringVar(&confFile, "config", "../etc/config.yaml", "Configure file")
	flag.Parse()

	if confFile != "" {
		ConfigFile = confFile
	}
	if showVer {
		fmt.Printf("Build name:\t%s\n", BuildName)
		fmt.Printf("Build version:\t%s\n", BuildVersion)
		fmt.Printf("Built time:\t%s\n", BuiltTime)
		fmt.Printf("Commit ID:\t%s\n", CommitID)
	}
}
