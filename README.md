# sql_exporter

## Overview

sql_exporter generates basic metrics for SQL result sets and expose them as Prometheus metrics.

Pros:
- Configuration driven application
- Logger included for debugging purpose
- Support integer and float metrics
- Each query can be run at a different time interval

- Cons:
- Currently, only support the gauge metric type
- Only support SQL databases

## Usage

The depot contain a go.mod and a go.sum files. Build the binary by running the following command:
```
$ go build
```

Then run it from the command line:
```
$ ./sql_exporter
```

By default, the path to the configuration file is `./config.yaml`. You can change it using the `-config` flag:
```
$ ./sql_exporter -config myconfiguration.yaml
```

Use the `-help` flag to get help information:
```
Usage of ./sql_exporter:
  -config string
        path to the configuration file (default "config.yaml")
  -help
        display the help message
```

## Configuration

A sample configuration file for the application looks like this:
```
addr: ":8080"
databases:
  - address: localhost:3306
      user: web_admin
      password:
      name: website_information
      metrics:
        - statement: "SELECT COUNT(VISIT_ID) FROM VISITS;"
          interval: 60m
          name: "Number of visits"
          help: "Count how many pages have been visited on the website."
          labels:
            data_source: "analytics"
```

List of available parameters and of their purpose:
- `addr` is the port on which the metrics will be exposed (by default, the port is :8080)
- `databases`  is the list of databases to connect to
- `address`, `name`, `user` and `password` are the database's information
- `statement`  is the mysql query to perform
- `interval` is the minimal interval between each query run
- `name`  is the name of the Prometheus gauge to spawn
- `labels`  are the Prometheus labels corresponding to the statement
