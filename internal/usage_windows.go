//go:build windows
// +build windows

package internal

const Usage = `Circonus Unified Agent, The plugin-driven server agent for collecting and reporting metrics.

Usage:

  circonus-unified-agentd.exe [commands|flags]

The commands & flags are:

  config              print out full sample configuration to stdout
  version             print the version to stdout

  --aggregator-filter <filter>   filter the aggregators to enable, separator is :
  --config <file>                configuration file to load
  --config-directory <directory> directory containing additional *.conf files
  --debug                        turn on debug logging
  --input-filter <filter>        filter the inputs to enable, separator is :
  --input-list                   print available input plugins.
  --output-filter <filter>       filter the outputs to enable, separator is :
  --output-list                  print available output plugins.
  --pidfile <file>               file to write our pid to
  --pprof-addr <address>         pprof address to listen on, don't activate pprof if empty
  --processor-filter <filter>    filter the processors to enable, separator is :
  --quiet                        run in quiet mode
  --sample-config                print out full sample configuration
  --section-filter               filter config sections to output, separator is :
                                 Valid values are 'agent', 'global_tags', 'outputs',
                                 'processors', 'aggregators' and 'inputs'
  --once                         enable once mode: gather metrics once, write them, and exit
  --test                         enable test mode: gather metrics once and print them
  --test-wait                    wait up to this many seconds for service
                                 inputs to complete in test or once mode
  --usage <plugin>               print usage for a plugin, ie, 'circonus-unified-agentd --usage mysql'
  --version                      display the version and exit

  --console                      run as console application (windows only)
  --service <service>            operate on the service (windows only)
  --service-name                 service name (windows only)
  --service-display-name         service display name (windows only)

Examples:

  # generate a config file:
  circonus-unified-agentd.exe config > circonus-unified-agent.conf

  # generate config with only cpu input & circonus output plugins defined
  circonus-unified-agentd.exe --input-filter cpu --output-filter circonus config

  # run a single collection, outputting metrics to stdout
  circonus-unified-agentd.exe --config circonus-unfied-agent.conf --test

  # run with all plugins defined in config file
  circonus-unified-agentd.exe --config circonus-unified-agent.conf

  # run, enabling the cpu & memory input, and circonus output plugins
  circonus-unified-agentd.exe --config circonus-unified-agent.conf --input-filter cpu:mem --output-filter circonus

  # run with pprof
  circonus-unified-agentd.exe --config circonus-unified-agent.conf --pprof-addr localhost:6060

  # run without service controller
  circonus-unified-agentd.exe --console install --config "C:\Program Files\Circonus\circonus-unified-agent.conf"

  # install as a service
  circonus-unified-agentd.exe --service install --config "C:\Program Files\Circonus\circonus-unified-agent.conf"

  # install as a service with custom name
  circonus-unified-agentd.exe --service install --service-name=my-circonus-unified-agent --service-display-name="MyCirconusUnifiedAgent"
`
