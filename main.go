package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/inconshreveable/log15"
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
	// Set a Command Line Interface.
	var (
		help       = flag.Bool("help", false, "display the help message")
		configPath = flag.String("config", "config.yaml", "path to the configuration file")
	)
	flag.Parse()
	if *help {
		flag.Usage()
		os.Exit(0)
	}

	log15.Info("sql_exporter started")

	// Load the configuration.
	config, err := loadConfiguration(*configPath)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Create a channel to gracefully shutdown the program.
	ctx, cancel := context.WithCancel(context.Background())
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, os.Kill)
	go func() {
		<-c
		fmt.Print("\n")
		log15.Info("interrupt signal received, monitorQuery() stopped")
		cancel()
	}()

	// Connect to the databases.
	var wg sync.WaitGroup
	for _, database := range config.Databases {
		log15.Info("connecting to the DB", "Address", database.Address, "DBName", database.Name)

		db, err := sqlx.Connect("mysql", (&mysql.Config{
			Addr:                 database.Address,
			Net:                  "tcp",
			User:                 database.User,
			Passwd:               database.Password,
			DBName:               database.Name,
			AllowNativePasswords: true,
		}).FormatDSN())
		if err != nil {
			log15.Crit(err.Error())
			os.Exit(1)
		}

		// Launch the queries.
		for _, query := range database.Queries {
			wg.Add(1)
			go func(db *sqlx.DB, query Query) {
				monitorQuery(ctx, db, query)
				wg.Done()
			}(db, query)
		}
	}
	wg.Wait()

	log15.Info("program gracefully shutdowned")
}

func loadConfiguration(path string) (c Configuration, err error) {
	log15.Info("loading the configuration", "configPath", path)

	raw, err := ioutil.ReadFile(path)
	if err != nil {
		log15.Crit(err.Error())
		return c, err
	}
	err = yaml.Unmarshal(raw, &c)
	if err != nil {
		log15.Crit(err.Error())
		return c, err
	}

	if len(c.Databases) == 0 {
		log15.Crit("no DB found inside the conf file", "configPath", path)
		return c, fmt.Errorf("no DB found inside the conf file: %s", path)
	}

	return c, nil
}

func monitorQuery(ctx context.Context, db *sqlx.DB, query Query) {
	t := time.NewTicker(query.Interval)

	var result float64
	var now time.Time
	for {
		select {
		case <-ctx.Done():
			return
		case now = <-t.C:
		}

		err := db.GetContext(ctx, &result, query.Statement)
		if err != nil {
			log15.Warn(err.Error())
			return
		}

		log15.Info("query executed", "timeStamp", now, "metric", query.Metric, "result", result)
	}
}
