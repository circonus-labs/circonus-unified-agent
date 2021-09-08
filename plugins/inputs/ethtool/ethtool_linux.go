//go:build linux
// +build linux

package ethtool

import (
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/circonus-labs/circonus-unified-agent/cua"
	"github.com/circonus-labs/circonus-unified-agent/filter"
	"github.com/circonus-labs/circonus-unified-agent/plugins/inputs"
	"github.com/safchain/ethtool"
)

type CommandEthtool struct {
	ethtool *ethtool.Ethtool
}

func (e *Ethtool) Gather(ctx context.Context, acc cua.Accumulator) error {

	// Get the list of interfaces
	interfaces, err := e.command.Interfaces()
	if err != nil {
		acc.AddError(err)
		return nil
	}

	interfaceFilter, err := filter.NewIncludeExcludeFilter(e.InterfaceInclude, e.InterfaceExclude)
	if err != nil {
		return fmt.Errorf("ethtool new filters: %w", err)
	}

	// parallelize the ethtool call in event of many interfaces
	var wg sync.WaitGroup

	for _, iface := range interfaces {

		// Check this isn't a loop back and that its matched by the filter
		if (iface.Flags&net.FlagLoopback == 0) && interfaceFilter.Match(iface.Name) {
			wg.Add(1)

			go func(i net.Interface) {
				e.gatherEthtoolStats(i, acc)
				wg.Done()
			}(iface)
		}
	}

	// Waiting for all the interfaces
	wg.Wait()
	return nil
}

// Initialise the Command Tool
func (e *Ethtool) Init() error {
	return e.command.Init()
}

// Gather the stats for the interface.
func (e *Ethtool) gatherEthtoolStats(iface net.Interface, acc cua.Accumulator) {

	tags := make(map[string]string)
	tags[tagInterface] = iface.Name

	driverName, err := e.command.DriverName(iface.Name)
	if err != nil {
		driverErr := fmt.Errorf("%s driver: %w", iface.Name, err)
		acc.AddError(driverErr)
		return
	}

	tags[tagDriverName] = driverName

	fields := make(map[string]interface{})
	stats, err := e.command.Stats(iface.Name)
	if err != nil {
		statsErr := fmt.Errorf("%s stats: %w", iface.Name, err)
		acc.AddError(statsErr)
		return
	}

	for k, v := range stats {
		fields[k] = v
	}

	acc.AddFields(pluginName, fields, tags)
}

func NewCommandEthtool() *CommandEthtool {
	return &CommandEthtool{}
}

func (c *CommandEthtool) Init() error {

	if c.ethtool != nil {
		return nil
	}

	e, err := ethtool.NewEthtool()
	if err != nil {
		return fmt.Errorf("ethtool init: %w", err)
	}

	c.ethtool = e

	return nil
}

func (c *CommandEthtool) DriverName(intf string) (string, error) {
	return c.ethtool.DriverName(intf)
}

func (c *CommandEthtool) Stats(intf string) (map[string]uint64, error) {
	return c.ethtool.Stats(intf)
}

func (c *CommandEthtool) Interfaces() ([]net.Interface, error) {

	// Get the list of interfaces
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("ethtool interfaces: %w", err)
	}

	return interfaces, nil
}

func init() {

	inputs.Add(pluginName, func() cua.Input {
		return &Ethtool{
			InterfaceInclude: []string{},
			InterfaceExclude: []string{},
			command:          NewCommandEthtool(),
		}
	})
}
