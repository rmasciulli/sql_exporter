package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/inconshreveable/log15"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	yaml "gopkg.in/yaml.v2"
)

type Configuration struct {
	Addr      string     `yaml:"addr"`
	Databases []Database `yaml:"databases"`
}

type Database struct {
	Address  string   `yaml:"address"`
	User     string   `yaml:"user"`
	Password string   `yaml:"password"`
	Name     string   `yaml:"name"`
	Metrics  []Metric `yaml:"metrics"`
}

type Metric struct {
	Statement string            `yaml:"statement"`
	Interval  time.Duration     `yaml:"interval"`
	Name      string            `yaml:"name"`
	Help      string            `yaml:"help"`
	Labels    prometheus.Labels `yaml:"labels"`
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
		for _, metric := range database.Metrics {
			wg.Add(1)
			go func(db *sqlx.DB, metric Metric) {
				log15.Info("monitoring metric", "name", metric.Name, "database", database.Name)
				monitorMetric(ctx, db, metric)
				log15.Info("stopped monitoring metric", "name", metric.Name, "database", database.Name)
				wg.Done()
			}(db, metric)
		}
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	srv := &http.Server{
		Handler: mux,
		Addr:    config.Addr,
	}

	go func() {
		<-ctx.Done()
		ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
		err := srv.Shutdown(ctx)
		if err != nil {
			log15.Error("shutting down http server", "error", err.Error())
		}
	}()

	err = srv.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		log15.Crit("starting http server", "error", err.Error())
		cancel()
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

	if len(c.Addr) == 0 {
		c.Addr = ":8080"
	}

	return c, nil
}

func monitorMetric(ctx context.Context, db *sqlx.DB, metric Metric) {
	t := time.NewTicker(metric.Interval)
	log15.Info("initialize metric", "name", metric.Name)

	gauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name:        metric.Name,
		Help:        metric.Help,
		ConstLabels: metric.Labels,
	})
	err := prometheus.Register(gauge)
	if err != nil {
		log15.Error("registering metric", "name", metric.Name, "error", err.Error())
		return
	}

	var value float64
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}

		err := db.GetContext(ctx, &value, metric.Statement)
		if err != nil {
			log15.Error("executing statement", "error", err.Error())
			return
		}

		gauge.Set(value)
		log15.Debug("evaluated metric", "name", metric.Name, "value", value)
	}
}
