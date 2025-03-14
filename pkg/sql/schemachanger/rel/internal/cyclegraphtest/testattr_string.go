// Copyright 2023 The Cockroach Authors.
//
// Use of this software is governed by the CockroachDB Software License
// included in the /LICENSE file.

// Code generated by "stringer"; DO NOT EDIT.

package cyclegraphtest

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[s-0]
	_ = x[s1-1]
	_ = x[s2-2]
	_ = x[c-3]
	_ = x[name-4]
}

func (i testAttr) String() string {
	switch i {
	case s:
		return "s"
	case s1:
		return "s1"
	case s2:
		return "s2"
	case c:
		return "c"
	case name:
		return "name"
	default:
		return "testAttr(" + strconv.FormatInt(int64(i), 10) + ")"
	}
}
