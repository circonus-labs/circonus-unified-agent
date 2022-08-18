//go:build !windows
// +build !windows

package internal

const Usage = `The plugin-driven server agent for collecting and reporting metrics.

Usage:

  circonus-unified-agent [commands|flags]

The commands & flags are:

  config              print out full sample configuration to stdout
  version             print the version to stdout

  --aggregator-filter <filter>   filter the aggregators to enable, separator is :
  --config <file>                configuration file to load
  --config-directory <directory> directory containing additional *.conf files
  --plugin-directory             directory containing *.so files, this directory will be
                                 searched recursively. Any Plugin found will be loaded
                                 and namespaced.
  --debug                        turn on debug logging
  --input-filter <filter>        filter the inputs to enable, separator is :
  --input-list                   print available input plugins.
  --output-filter <filter>       filter the outputs to enable, separator is :
  --output-list                  print available output plugins.
  --pidfile <file>               file to write our pid to
  --pprof-addr <address>         pprof address to listen on, don't activate pprof if empty
  --processor-filter <filter>    filter the processors to enable, separator is :
  --quiet                        run in quiet mode
  --section-filter               filter config sections to output, separator is :
                                 Valid values are 'agent', 'global_tags', 'outputs',
                                 'processors', 'aggregators' and 'inputs'
  --sample-config                print out full sample configuration
  --once                         enable once mode: gather metrics once, write them, and exit
  --test                         enable test mode: gather metrics once and print them
  --test-wait                    wait up to this many seconds for service
                                 inputs to complete in test or once mode
  --usage <plugin>               print usage for a plugin, ie, 'circonus-unified-agent --usage mysql'
  --version                      display the version and exit

Examples:

  # generate a config file:
  circonus-unified-agent config > circonus-unified-agent.conf

  # generate config with only cpu input & circonus output plugins defined
  circonus-unified-agent --input-filter cpu --output-filter circonus config

  # run a single collection, outputting metrics to stdout
  circonus-unified-agent --config circonus-unified-agent.conf --test

  # run with all plugins defined in config file
  circonus-unified-agent --config circonus-unified-agent.conf

  # run, enabling the cpu & memory input, and circonus output plugins
  circonus-unified-agent --config circonus-unified-agent.conf --input-filter cpu:mem --output-filter circonus

  # run with pprof
  circonus-unified-agent --config circonus-unified-agent.conf --pprof-addr localhost:6060
`
