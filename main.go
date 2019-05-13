package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"gopkg.in/yaml.v2"
)

func main() {
	var (
		help       = flag.Bool("help", false, "display the help message")
		configPath = flag.String("config", "config.yaml", "path to the configuration file")
	)

	flag.Parse()

	if *help {
		flag.Usage()
		os.Exit(0)
	}

	var config struct {
	}

	raw, err := ioutil.ReadFile(*configPath)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	err = yaml.Unmarshal(raw, &config)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Println("sql_exporter")
}
