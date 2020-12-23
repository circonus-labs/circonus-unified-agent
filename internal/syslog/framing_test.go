package syslog

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFraming(t *testing.T) {
	var f1 Framing
	_ = f1.UnmarshalTOML([]byte(`"non-transparent"`))
	assert.Equal(t, NonTransparent, f1)

	var f2 Framing
	_ = f2.UnmarshalTOML([]byte(`non-transparent`))
	assert.Equal(t, NonTransparent, f2)

	var f3 Framing
	_ = f3.UnmarshalTOML([]byte(`'non-transparent'`))
	assert.Equal(t, NonTransparent, f3)

	var f4 Framing
	_ = f4.UnmarshalTOML([]byte(`"octet-counting"`))
	assert.Equal(t, OctetCounting, f4)

	var f5 Framing
	_ = f5.UnmarshalTOML([]byte(`octet-counting`))
	assert.Equal(t, OctetCounting, f5)

	var f6 Framing
	_ = f6.UnmarshalTOML([]byte(`'octet-counting'`))
	assert.Equal(t, OctetCounting, f6)

	var f7 Framing
	err := f7.UnmarshalTOML([]byte(`nope`))
	assert.Equal(t, Framing(-1), f7)
	assert.Error(t, err)
}
