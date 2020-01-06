package testcat

type NodeInfo struct {
	NodeID   int32          `json:"node_id"`
	Locality string         `json:"locality,omitempty"`
	Activity []JSONActivity `json:"activity,omitempty"`
}

type JSONActivity struct {
	NodeID   int32 `json:"node_id"`
	Incoming int64 `json:"incoming"`
	Outgoing int64 `json:"outgoing"`
	Latency  int64 `json:"latency"`
}
