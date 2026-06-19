package ingest

import "strings"

// canonProject normalizes a project name for grouping: upper-cased, collapsed
// whitespace, with the volatile " EXT"/version suffix folded so "VERLIM 3 EXT"
// and "VERLIM 3" aggregate together. Display uses Title-case via titleCase.
func canonProject(s string) string {
	s = strings.ToUpper(strings.TrimSpace(s))
	s = strings.Join(strings.Fields(s), " ")
	s = strings.TrimSuffix(s, " EXT")
	return s
}

// normGP normalizes the GP group label ("gp 3" → "GP 3").
func normGP(s string) string {
	s = strings.ToUpper(strings.TrimSpace(s))
	return strings.Join(strings.Fields(s), " ")
}

// canonSales normalizes a sales/agent name, dropping the leading "AGENT"/"NON
// SALES"/"MANAJEMEN" qualifier into a clean display name (Title-case).
func canonSales(s string) string {
	s = strings.Join(strings.Fields(strings.TrimSpace(s)), " ")
	if s == "" {
		return "—"
	}
	return titleCase(s)
}

// isAgent reports whether a sales label denotes an external agent.
func isAgent(name string) bool {
	u := strings.ToUpper(name)
	return strings.Contains(u, "AGENT") || strings.Contains(u, "AGEN ")
}

// normBank canonicalizes a bank name; cash/blank deals collapse to "" (no bank).
func normBank(s string) string {
	u := strings.ToUpper(strings.TrimSpace(s))
	u = strings.Join(strings.Fields(u), " ")
	switch {
	case u == "", u == "-", u == "CASH", strings.HasPrefix(u, "CASH"):
		return ""
	}
	return u
}

// normCara classifies a payment scheme into KPR / Cash Keras / Cash Bertahap.
func normCara(s string) string {
	u := strings.ToUpper(strings.TrimSpace(s))
	switch {
	case strings.Contains(u, "KPR"):
		return "KPR"
	case strings.Contains(u, "BERTAHAP"):
		return "Cash Bertahap"
	case strings.Contains(u, "KERAS"), strings.Contains(u, "CASH"):
		return "Cash Keras"
	case u == "":
		return "Lainnya"
	default:
		return titleCase(u)
	}
}

// monthOrder maps an Indonesian month (any casing, with/without year suffix) to
// its 1..12 index and short label.
var monthLong = []string{
	"januari", "februari", "maret", "april", "mei", "juni",
	"juli", "agustus", "september", "oktober", "november", "desember",
}
var monthShort = []string{
	"Jan", "Feb", "Mar", "Apr", "Mei", "Jun", "Jul", "Agu", "Sep", "Okt", "Nov", "Des",
}

// monthIndex returns 1..12 for a month cell, or 0 if unrecognized.
func monthIndex(s string) int {
	u := strings.ToLower(strings.TrimSpace(s))
	for i, m := range monthLong {
		if strings.Contains(u, m) || strings.HasPrefix(u, m[:3]) {
			return i + 1
		}
	}
	return 0
}

// normMonth returns the canonical short month label ("Jan"), or "" if unknown.
func normMonth(s string) string {
	if i := monthIndex(s); i > 0 {
		return monthShort[i-1]
	}
	return ""
}

// titleCase upper-cases the first letter of each word, lower-casing the rest.
func titleCase(s string) string {
	words := strings.Fields(strings.ToLower(s))
	for i, w := range words {
		r := []rune(w)
		if len(r) > 0 {
			r[0] = []rune(strings.ToUpper(string(r[0])))[0]
		}
		words[i] = string(r)
	}
	return strings.Join(words, " ")
}
