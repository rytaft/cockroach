package cat

import (
	"fmt"

	"github.com/cockroachdb/cockroach/pkg/roachpb"
	"github.com/cockroachdb/cockroach/pkg/sql/sem/tree"
)

type Partitioning struct {
	Ranges []Range
}

type Range struct {
	From   tree.Datums
	To     tree.Datums
	NodeID roachpb.NodeID
	Zone   Zone
}

func (r *Range) String() string {
	return fmt.Sprintf("from:%s to:%s node:%s", r.From, r.To, r.NodeID)
}
