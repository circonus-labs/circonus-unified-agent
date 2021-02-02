package riemannlistener

import (
	"log"
	"testing"
	"time"

	"github.com/circonus-labs/circonus-unified-agent/internal"
	"github.com/circonus-labs/circonus-unified-agent/testutil"
	riemanngo "github.com/riemann/riemann-go-client"
	"github.com/stretchr/testify/require"
	"gotest.tools/assert"
)

func TestSocketListener_tcp(t *testing.T) {
	log.Println("Entering")

	sl := newRiemannSocketListener()
	sl.Log = testutil.Logger{}
	sl.ServiceAddress = "tcp://127.0.0.1:5555"
	sl.ReadBufferSize = internal.Size{Size: 1024}

	acc := &testutil.Accumulator{}
	err := sl.Start(acc)
	require.NoError(t, err)
	defer sl.Stop()

	testStats(t, sl)
	testMissingService(t, sl)
}
func testStats(t *testing.T, sl *RiemannSocketListener) {
	c := riemanngo.NewTCPClient(sl.ServiceAddress, 5*time.Second)
	err := c.Connect()
	if err != nil {
		log.Println("Error")
		panic(err)
	}
	defer c.Close()
	result, _ := riemanngo.SendEvent(c, &riemanngo.Event{
		Service: "hello",
	})
	assert.Equal(t, result.GetOk(), true)

}
func testMissingService(t *testing.T, sl *RiemannSocketListener) {
	c := riemanngo.NewTCPClient(sl.ServiceAddress, 5*time.Second)
	err := c.Connect()
	if err != nil {
		panic(err)
	}
	defer c.Close()
	result, _ := riemanngo.SendEvent(c, &riemanngo.Event{})
	assert.Equal(t, result.GetOk(), false)

}
