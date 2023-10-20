package daemon

const (
	PriorityDisconnectINX = iota // no dependencies
	PriorityStopIndexer
	PriorityStopIndexerAcceptedTransactions
	PriorityStopIndexerAPI
	PriorityStopPrometheus
)
