//go:build windows
// +build windows

package main

import (
	"flag"
	"log"
	"os"
	"runtime"

	"github.com/circonus-labs/circonus-unified-agent/logger"
	"github.com/kardianos/service"
)

var fService = flag.String("service", "", "operate on the service (windows only)")
var fServiceName = flag.String("service-name", "circonus-unified-agent", "service name (windows only)")
var fServiceDisplayName = flag.String("service-display-name", "Circonus Unified Agent Data Collector Service", "service display name (windows only)")
var fRunAsConsole = flag.Bool("console", false, "run as console application (windows only)")

func run(inputFilters, outputFilters, aggregatorFilters, processorFilters []string) {
	if runtime.GOOS == "windows" && windowsRunAsService() {
		runAsWindowsService(
			inputFilters,
			outputFilters,
			aggregatorFilters,
			processorFilters,
		)
	} else {
		stop = make(chan struct{})
		reloadLoop(
			inputFilters,
			outputFilters,
			aggregatorFilters,
			processorFilters,
		)
	}
}

type program struct {
	inputFilters      []string
	outputFilters     []string
	aggregatorFilters []string
	processorFilters  []string
}

func (p *program) Start(s service.Service) error {
	go p.run()
	return nil
}
func (p *program) run() {
	stop = make(chan struct{})
	reloadLoop(
		p.inputFilters,
		p.outputFilters,
		p.aggregatorFilters,
		p.processorFilters,
	)
}
func (p *program) Stop(s service.Service) error {
	close(stop)
	return nil
}

func runAsWindowsService(inputFilters, outputFilters, aggregatorFilters, processorFilters []string) {
	programFiles := os.Getenv("ProgramFiles")
	if programFiles == "" {
		programFiles = "C:\\Program Files"
		log.Print("I! ProgramFiles environment variable is unset")
	} else {
		log.Print("I! ProgramFiles found with value: " + programFiles)
	}
	svcConfig := &service.Config{
		Name:        *fServiceName,
		DisplayName: *fServiceDisplayName,
		Description: "Collects data using a series of plugins and publishes it to " +
			"another series of plugins.",
		Arguments: []string{"--config", programFiles + `\Circonus Unified Agent\etc\circonus-unified-agent.conf`},
	}

	prg := &program{
		inputFilters:      inputFilters,
		outputFilters:     outputFilters,
		aggregatorFilters: aggregatorFilters,
		processorFilters:  processorFilters,
	}
	s, err := service.New(prg, svcConfig)
	if err != nil {
		log.Fatal("E! " + err.Error())
	}
	// Handle the --service flag here to prevent any issues with tooling that
	// may not have an interactive session, e.g. installing from Ansible.
	if *fService != "" {
		if *fConfig != "" {
			svcConfig.Arguments = []string{"--config", *fConfig}
		}
		if *fConfigDirectory != "" {
			svcConfig.Arguments = append(svcConfig.Arguments, "--config-directory", *fConfigDirectory)
		}
		// set servicename to service cmd line, to have a custom name after relaunch as a service
		svcConfig.Arguments = append(svcConfig.Arguments, "--service-name", *fServiceName)

		err := service.Control(s, *fService)
		if err != nil {
			log.Fatal("E! " + err.Error())
		}
		os.Exit(0)
	} else {
		winlogger, err := s.Logger(nil)
		if err == nil {
			// When in service mode, register eventlog target and setup default logging to eventlog
			logger.RegisterEventLogger(winlogger)
			logger.SetupLogging(logger.LogConfig{LogTarget: logger.LogTargetEventlog})
		}
		err = s.Run()

		if err != nil {
			log.Println("E! " + err.Error())
		}
	}
}

// Return true if agent should create a Windows service.
func windowsRunAsService() bool {
	if *fService != "" {
		return true
	}

	if *fRunAsConsole {
		return false
	}

	return !service.Interactive()
}
