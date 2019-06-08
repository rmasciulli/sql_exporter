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
	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"
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
	log15.Info("loading configuration", "path", *configPath)
	config, err := loadConfiguration(*configPath)
	if err != nil {
		log15.Crit("loading configuration", "error", err.Error())
		os.Exit(1)
	}

	// Create a channel to gracefully shutdown the program.
	ctx, cancel := context.WithCancel(context.Background())
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, os.Kill)
	go func() {
		<-c
		log15.Info("interrupt signal received")
		cancel()
	}()

	// Connect to the databases.
	var wg sync.WaitGroup
	for _, database := range config.Databases {
		log15.Info("connecting to database", "address", database.Address, "database", database.Name)

		db, err := sqlx.Connect("mysql", (&mysql.Config{
			Addr:                 database.Address,
			Net:                  "tcp",
			User:                 database.User,
			Passwd:               database.Password,
			DBName:               database.Name,
			AllowNativePasswords: true,
		}).FormatDSN())
		if err != nil {
			log15.Crit("connecting to database", "error", err.Error())
			os.Exit(1)
		}

		// Launch the queries.
		for _, query := range database.Queries {
			wg.Add(1)
			go func(db *sqlx.DB, query Query) {
				log15.Info("monitoring query", "metric", query.Metric, "database", database.Name)
				monitorQuery(ctx, db, query)
				log15.Info("stopped monitoring query", "metric", query.Metric, "database", database.Name)
				wg.Done()
			}(db, query)
		}
	}
	wg.Wait()

	log15.Info("program gracefully shutdown")
}

func loadConfiguration(path string) (c Configuration, err error) {
	raw, err := ioutil.ReadFile(path)
	if err != nil {
		return c, errors.Wrap(err, "opening file")
	}
	err = yaml.Unmarshal(raw, &c)
	if err != nil {
		return c, errors.Wrap(err, "parsing file")
	}

	if len(c.Databases) == 0 {
		return c, fmt.Errorf("no database found")
	}

	return c, nil
}

func monitorQuery(ctx context.Context, db *sqlx.DB, query Query) {
	t := time.NewTicker(query.Interval)

	var result float64
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}

		err := db.GetContext(ctx, &result, query.Statement)
		if err != nil {
			log15.Error("executing query", "error", err.Error())
			return
		}

		log15.Info("query executed", "metric", query.Metric, "result", result)
	}
}
