package daemon

const (
	PriorityDisconnectINX = iota // no dependencies
	PriorityStopIndexer
	PriorityStopIndexerAPI
	PriorityStopPrometheus
)
