package leases

import (
	"time"

	apierrors "rms-be/internal/api/errors"
)

// endExclusiveUpper returns an upper bound for half-open interval [start, end) overlap checks.
// A nil end date means “open-ended” (no finite end).
func endExclusiveUpper(end *time.Time) time.Time {
	if end == nil {
		return time.Date(9999, 12, 31, 23, 59, 59, 999999999, time.UTC)
	}
	return *end
}

// RangesOverlap reports whether two half-open ranges [aStart, aEnd) and [bStart, bEnd) intersect.
// Nil end dates mean an open-ended upper bound.
func RangesOverlap(aStart time.Time, aEnd *time.Time, bStart time.Time, bEnd *time.Time) bool {
	aU := endExclusiveUpper(aEnd)
	bU := endExclusiveUpper(bEnd)
	return aStart.Before(bU) && bStart.Before(aU)
}

// ValidateLeaseDateOrder ensures end is strictly after start when present.
func ValidateLeaseDateOrder(start time.Time, end *time.Time) error {
	if end == nil {
		return nil
	}
	if !end.After(start) {
		return apierrors.ErrValidation
	}
	return nil
}

// OverlapsActivePrimaryLease returns true if candidate overlaps any existing lease in others
// (each expected to be active + primary on the same unit; caller filters).
func OverlapsActivePrimaryLease(candidateStart time.Time, candidateEnd *time.Time, others []Lease) bool {
	for i := range others {
		o := &others[i]
		if o.Status != LeaseStatusActive || !o.IsPrimary {
			continue
		}
		if RangesOverlap(candidateStart, candidateEnd, o.StartDate, o.EndDate) {
			return true
		}
	}
	return false
}
