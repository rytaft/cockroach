package opt

import (
	"github.com/cockroachdb/cockroach/pkg/roachpb"
	"github.com/cockroachdb/cockroach/pkg/server/status/statuspb"
)

type NodeMeta struct {
	MetaID roachpb.NodeID

	Locality roachpb.Locality

	Activity map[roachpb.NodeID]statuspb.NodeStatus_NetworkActivity
}
