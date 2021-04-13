package snmp

import (
	"fmt"

	"github.com/soniah/gosnmp"
)

// provides capability to do automatic lookups on vendor-specific
// values which are integers that translate to strings for tags

func shouldLookupField(oid string) bool {
	if oid == "" {
		return false
	}
	keys := []string{
		".1.3.6.1.4.1.1991.1.1.1.1.25.0", // FOUNDRY-SN-AGENT-MIB::snChasArchitectureType.0
	}
	for _, key := range keys {
		if oid == key {
			return true
		}
	}
	return false
}

func fieldLookup(key string, sv gosnmp.SnmpPDU) (interface{}, error) {
	switch key {
	case ".1.3.6.1.4.1.1991.1.1.1.1.25.0":
		return foundrySnChasArchitectureType(sv.Value.(int)), nil
	default:
		return "", fmt.Errorf("unknown lookup key (%s)", key)
	}
}

func foundrySnChasArchitectureType(at int) string {
	switch at {
	case 1:
		return "stackable"
	case 2:
		return "bigIron"
	case 3:
		return "terathon"
	case 4:
		return "fifthGen"
	default:
		return "unknown"
	}
}
