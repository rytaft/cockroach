package physical

import (
	"github.com/cockroachdb/cockroach/pkg/sql/opt/constraint"
	"github.com/cockroachdb/cockroach/pkg/util"
)

type Partitioning struct {
	Ranges []Range
}

type Range struct {
	Span             constraint.Span
	Nodes            util.FastIntSet
	LeasePreferences util.FastIntSet
}
