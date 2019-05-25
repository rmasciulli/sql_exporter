package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"sync"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"gopkg.in/yaml.v2"
)

type Configuration struct {
	Databases []Database `yaml:"databases"`
}

type Database struct {
	Address  string  `yaml:"address"`
	User     string  `yaml:"user"`
	Password string  `yaml:"password"`
	Name     string  `yaml:"name"`
	Queries  []Query `yaml:"queries"`
}

type Query struct {
	Statement string        `yaml:"statement"`
	Interval  time.Duration `yaml:"interval"`
	Metric    string        `yaml:"metric"`
}

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

	config, err := loadConfiguration(*configPath)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	var wg sync.WaitGroup
	for _, database := range config.Databases {
		db, err := sqlx.Connect("mysql", (&mysql.Config{
			Addr:                 database.Address,
			Net:                  "tcp",
			User:                 database.User,
			Passwd:               database.Password,
			DBName:               database.Name,
			AllowNativePasswords: true,
		}).FormatDSN())
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		for _, query := range database.Queries {
			wg.Add(1)
			go func(db *sqlx.DB, query Query) {
				monitorQuery(db, query)
				wg.Done()
			}(db, query)
		}
	}
	wg.Wait()
}

func loadConfiguration(path string) (c Configuration, err error) {
	raw, err := ioutil.ReadFile(path)
	if err != nil {
		return c, err
	}
	err = yaml.Unmarshal(raw, &c)
	if err != nil {
		return c, err
	}

	if len(c.Databases) == 0 {
		return c, fmt.Errorf("no database found inside the config file: %s", path)
	}

	return c, nil
}

func monitorQuery(db *sqlx.DB, query Query) {
	t := time.NewTicker(query.Interval)

	var result float64
	for {
		now := <-t.C

		err := db.Get(&result, query.Statement)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		fmt.Println(now, query.Metric, result)
	}
}
