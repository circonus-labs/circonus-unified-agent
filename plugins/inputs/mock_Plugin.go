package inputs

import (
	"fmt"

	"github.com/circonus-labs/circonus-unified-agent/cua"
	"github.com/stretchr/testify/mock"
)

// MockPlugin struct should be named the same as the Plugin
type MockPlugin struct {
	mock.Mock
}

// Description will appear directly above the plugin definition in the config file
func (m *MockPlugin) Description() string {
	return `This is an example plugin`
}

// SampleConfig will populate the sample configuration portion of the plugin's configuration
func (m *MockPlugin) SampleConfig() string {
	return `  
  instance_id = "" # unique instance identifier (REQUIRED)

  sampleVar = 'foo'
`
}

// Gather defines what data the plugin will gather.
func (m *MockPlugin) Gather(_a0 cua.Accumulator) error {
	ret := m.Called(_a0)

	r0 := ret.Error(0)

	return fmt.Errorf("mock call: %w", r0)
}
