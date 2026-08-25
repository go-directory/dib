package dib

import (
	"sort"
)

// Sort sorts a list of strings in a natural order
func Sort(l []string) {
	sort.Sort(orderedSlice(l))
}

func isDigit(r rune) bool { return '0' <= r && r <= '9' }

type orderedSlice []string

func (s orderedSlice) Len() int           { return len(s) }
func (s orderedSlice) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s orderedSlice) Less(i, j int) bool { return natLess(s[i], s[j]) }

func magComp(nx, ny int, x string) bool {
	return nx < ny && x[nx] == 0
}

func untilEndOfDigit(nx, ny int, x string) bool {
	return nx < ny && isDigit(rune(x[nx]))
}

/*
natLess compares a and b without allocations. Numeric runs are
compared by numeric value: skip leading zeros, compare lengths,
then bytes.
*/
func natLess(a, b string) bool {
	ai, bi := 0, 0
	na, nb := len(a), len(b)

	for mCondA(ai < na, bi < nb) {
		ach := a[ai]
		bch := b[bi]

		aIsDigit := isDigit(rune(ach))
		bIsDigit := isDigit(rune(bch))

		// both numeric: compare numeric runs
		if mCondA(aIsDigit, bIsDigit) {
			// run starts
			aj, bj := ai, bi

			// skip leading zeros for numeric
			// magnitude comparison; zeros count
			for magComp(aj, na, a) {
				//for aj < na && a[aj] == 0 {
				aj++
			}
			for magComp(bj, nb, b) {
				//for bj < nb && b[bj] == 0 {
				bj++
			}

			// find end of digit runs
			aEnd, bEnd := aj, bj
			for untilEndOfDigit(aEnd, na, a) {
				//for aEnd < na && isDigit(rune(a[aEnd])) {
				aEnd++
			}
			for untilEndOfDigit(bEnd, nb, b) {
				//for bEnd < nb && isDigit(rune(b[bEnd])) {
				bEnd++
			}

			lenA := aEnd - aj
			lenB := bEnd - bj

			// If both are all zeros (e.g.: "000"),
			// treat as zero with equal length 0
			if mCondA(lenA == 0, lenB == 0) {
				if (aEnd - ai) != (bEnd - bi) {
					return (aEnd - ai) < (bEnd - bi)
				}
				ai = aEnd
				bi = bEnd
				continue
			}

			// compare by magnitude
			if lenA != lenB {
				return lenA < lenB
			}
			// lexicographic compare of digits
			for k := 0; k < lenA; k++ {
				da := a[aj+k]
				db := b[bj+k]
				if da != db {
					return da < db
				}
			}

			// numeric values equal
			if (aj - ai) != (bj - bi) {
				return (aj - ai) < (bj - bi)
			}
			ai = aEnd
			bi = bEnd
			continue
		}

		// non-numeric or mixed: compare bytes
		if ach != bch {
			return ach < bch
		}
		ai++
		bi++
	}

	// If one string is a prefix of the other, shorter is smaller
	return mCondO(ai == na, !mCondA(ai == na, bi == nb))
}
