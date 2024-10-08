// Copyright 2017 The Cockroach Authors.
//
// Use of this software is governed by the CockroachDB Software License
// included in the /LICENSE file.

package lexbase

import (
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// Special case normalization rules for Turkish/Azeri lowercase dotless-i and
// uppercase dotted-i. Fold both dotted and dotless 'i' into the ascii i/I, so
// our case-insensitive comparison functions can be locale-invariant. This
// mapping implements case-insensitivity for Turkish and other latin-derived
// languages simultaneously, with the additional quirk that it is also
// insensitive to the dottedness of the i's
var normalize = unicode.SpecialCase{
	unicode.CaseRange{
		Lo: 0x0130,
		Hi: 0x0130,
		Delta: [unicode.MaxCase]rune{
			0x49 - 0x130, // Upper
			0x69 - 0x130, // Lower
			0x49 - 0x130, // Title
		},
	},
	unicode.CaseRange{
		Lo: 0x0131,
		Hi: 0x0131,
		Delta: [unicode.MaxCase]rune{
			0x49 - 0x131, // Upper
			0x69 - 0x131, // Lower
			0x49 - 0x131, // Title
		},
	},
}

// NormalizeName normalizes to lowercase and Unicode Normalization
// Form C (NFC).
func NormalizeName(n string) string {
	lower := strings.Map(normalize.ToLower, n)
	if isASCII(lower) {
		return lower
	}
	return norm.NFC.String(lower)
}

// NormalizeString normalizes to Unicode Normalization Form C (NFC).
// This function is specifically for double quoted identifiers.
func NormalizeString(s string) string {
	if isASCII(s) {
		return s
	}
	return norm.NFC.String(s)
}
