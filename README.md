# sql_exporter

## Overview

sql_exporter generates basic metrics for SQL result sets and expose them as Prometheus metrics.

Pros:
- Configuration driven application
- Logger included for debugging purpose
- Support integer and float metrics
- Each query can be run at a different time interval

Cons:
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

Here is a sample configuration describing the default values with comments:

```
# The address on which the server will list (interface:port)
addr: ":8080"

# One entry per database you want to connect to. Databases 
# and metrics are handled in parallel.
databases:

#    # Address of the database (address:port)
#  - address: localhost:3306
#    user: web_admin
#    password:
#    name: website_information

#    # List of metrics to retrieve for this database
#    metrics:

#        # Statement to execute
#      - statement: "select count(visit_id) from visits"
#        # Interval at which the statement is executed. If the
#        # statement takes more than the interval duration to
#        # run, the next execution will start right away.
#        interval: 60m
#        # Name of the metric to expose.
#        name: "namespace_subsystem_name_unit"
#        # Help message of the metric.
#        help: "number of visits on the website"
#        # Labels to add to the metric.
#        labels:
#          data_source: "analytics"
```

## sql_exporter as a service

To expose your metrics continuously, you can run sql_exporter as a service.
You can use `sql_exporter.service` as a template to create your own service configuration:

```
# /usr/lib/systemd/system/sql_exporter.service
[Unit]
Description=SQL Exporter, Prometheus exporter that collect metrics from SQL databases
Documentation=https://github.com/rmasciulli/sql_exporter/blob/master/README.md
After=network-online.target

[Service]
Type=simple
ExecStart=/opt/sql_exporter/sql_exporter -conf config.yaml
KillSignal=SIGINT
KillMode=process
Restart=on-failure

[Install]
WantedBy=multi-user.target
```
