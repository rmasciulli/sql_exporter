package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	var (
		help = flag.Bool("help", false, "display the help message")
	)

	flag.Parse()

	if *help {
		flag.Usage()
		os.Exit(0)
	}

	fmt.Println("sql_exporter")
}
