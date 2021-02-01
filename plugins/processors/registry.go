package processors

import "github.com/circonus-labs/circonus-unified-agent/cua"

type Creator func() cua.Processor
type StreamingCreator func() cua.StreamingProcessor

// all processors are streaming processors.
// cua.Processor processors are upgraded to cua.StreamingProcessor

var Processors = map[string]StreamingCreator{}

// Add adds a cua.Processor processor
func Add(name string, creator Creator) {
	Processors[name] = upgradeToStreamingProcessor(creator)
}

// AddStreaming adds a cua.StreamingProcessor streaming processor
func AddStreaming(name string, creator StreamingCreator) {
	Processors[name] = creator
}

func upgradeToStreamingProcessor(oldCreator Creator) StreamingCreator {
	return func() cua.StreamingProcessor {
		return NewStreamingProcessorFromProcessor(oldCreator())
	}
}
