package snmp

import (
	"github.com/circonus-labs/circonus-unified-agent/internal"
)

// type ClientConfig struct {
// 	// Timeout to wait for a response.
// 	Timeout internal.Duration `toml:"timeout"`
// 	Retries int               `toml:"retries"`
// 	// Values: 1, 2, 3
// 	Version uint8 `toml:"version"`

// 	// Parameters for Version 1 & 2
// 	Community string `toml:"community"`

// 	// Parameters for Version 2 & 3
// 	MaxRepetitions uint32 `toml:"max_repetitions"`

// 	// Parameters for Version 3
// 	ContextName string `toml:"context_name"`
// 	// Values: "noAuthNoPriv", "authNoPriv", "authPriv"
// 	SecLevel string `toml:"sec_level"`
// 	SecName  string `toml:"sec_name"`
// 	// Values: "MD5", "SHA", "". Default: ""
// 	AuthProtocol string `toml:"auth_protocol"`
// 	AuthPassword string `toml:"auth_password"`
// 	// Values: "DES", "AES", "". Default: ""
// 	PrivProtocol string `toml:"priv_protocol"`
// 	PrivPassword string `toml:"priv_password"`
// 	EngineID     string `toml:"-"`
// 	EngineBoots  uint32 `toml:"-"`
// 	EngineTime   uint32 `toml:"-"`
// }

type ClientConfig struct {
	AuthProtocol   string            `toml:"auth_protocol"` // Values: "MD5", "SHA", "". Default: ""
	AuthPassword   string            `toml:"auth_password"`
	PrivProtocol   string            `toml:"priv_protocol"` // Values: "DES", "AES", "". Default: ""
	PrivPassword   string            `toml:"priv_password"`
	EngineID       string            `toml:"-"`
	Community      string            `toml:"community"`    // Parameters for Version 1 & 2
	ContextName    string            `toml:"context_name"` // Parameters for Version 3
	SecLevel       string            `toml:"sec_level"`    // Values: "noAuthNoPriv", "authNoPriv", "authPriv"
	SecName        string            `toml:"sec_name"`
	Timeout        internal.Duration `toml:"timeout"` // Timeout to wait for a response.
	Retries        int               `toml:"retries"`
	MaxRepetitions uint32            `toml:"max_repetitions"` // Parameters for Version 2 & 3
	EngineBoots    uint32            `toml:"-"`
	EngineTime     uint32            `toml:"-"`
	Version        uint8             `toml:"version"` // Values: 1, 2, 3
}
