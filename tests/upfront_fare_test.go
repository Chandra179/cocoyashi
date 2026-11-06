package tests

import (
	"errors"
	"testing"

	"cocoyashi/ride"
)

// charge runs a trip with the given facts and returns what the gateway captured.
func charge(t *testing.T, f ride.Facts) ride.Money {
	t.Helper()
	s, _, gw := newSvcFull(cfg())
	id := inProgress(t, s) // upfront fare 41000
	if err := s.CompleteWith(id, f); err != nil {
		t.Fatalf("complete: %v", err)
	}
	return gw.captured[string(id)]
}

// FARE-1 (b): tolls that were not in the estimate are added.
func TestTollsAreAdded(t *testing.T) {
	if got := charge(t, ride.Facts{Tolls: 6000}); got != 47000 {
		t.Fatalf("charged %d, want 47000 (41000 + 6000 tolls)", got)
	}
}

// FARE-1, forbidden: a longer trip inside the detour threshold, with no change by the rider, costs no more.
func TestSlowerTripWithinThresholdCostsTheUpfrontFare(t *testing.T) {
	f := ride.Facts{EstMinutes: 30, ActualMinutes: 35, Metered: 50000}
	if got := charge(t, f); got != 41000 {
		t.Fatalf("charged %d, want 41000", got)
	}
}

// FARE-1, forbidden: a detour the driver caused never raises the fare, however long it was.
func TestDriverDetourDoesNotRaiseTheFare(t *testing.T) {
	f := ride.Facts{Cause: ride.DriverDetour, EstMinutes: 30, ActualMinutes: 60, Metered: 70000}
	if got := charge(t, f); got != 41000 {
		t.Fatalf("charged %d, want 41000", got)
	}
}

// FARE-1 (e): a rider detour beyond the threshold is charged by actual time and distance.
func TestRiderDetourBeyondThresholdIsMetered(t *testing.T) {
	f := ride.Facts{Cause: ride.RiderDetour, EstMinutes: 30, ActualMinutes: 45, Metered: 52000}
	if got := charge(t, f); got != 52000 {
		t.Fatalf("charged %d, want 52000", got)
	}
}

// FARE-1 (e): a rider detour inside the threshold keeps the upfront fare.
func TestRiderDetourWithinThresholdKeepsUpfrontFare(t *testing.T) {
	f := ride.Facts{Cause: ride.RiderDetour, EstMinutes: 30, ActualMinutes: 38, Metered: 52000}
	if got := charge(t, f); got != 41000 {
		t.Fatalf("charged %d, want 41000 (38 min is 26%% over, threshold 30%%)", got)
	}
}

// FARE-1 (e): only a detour the rider caused can raise the fare; a slow trip with no cause cannot, even far past the threshold.
func TestLongTripWithNoRiderCauseCostsTheUpfrontFare(t *testing.T) {
	f := ride.Facts{Cause: ride.NoDetour, EstMinutes: 30, ActualMinutes: 60, Metered: 70000}
	if got := charge(t, f); got != 41000 {
		t.Fatalf("charged %d, want 41000", got)
	}
}

// FARE-1 (a), required: a destination change shows a new price and applies it once the rider confirms.
func TestConfirmedDestinationChangeIsCharged(t *testing.T) {
	s, _, gw := newSvcFull(cfg())
	id := inProgress(t, s)

	if err := s.ChangeDestination(id, quote(55000)); err != nil {
		t.Fatalf("change: %v", err)
	}
	if err := s.ConfirmPrice(id); err != nil {
		t.Fatalf("confirm: %v", err)
	}
	_ = s.Complete(id)

	if got := gw.captured[string(id)]; got != 55000 {
		t.Fatalf("charged %d, want 55000", got)
	}
}

// FARE-1 (a), forbidden: a new price the rider has not confirmed is not charged.
func TestUnconfirmedPriceChangeIsNotCharged(t *testing.T) {
	s, _, gw := newSvcFull(cfg())
	id := inProgress(t, s)

	_ = s.ChangeDestination(id, quote(55000))
	_ = s.Complete(id)

	if got := gw.captured[string(id)]; got != 41000 {
		t.Fatalf("charged %d, want 41000", got)
	}
}

// FARE-1 (a), required: the market allows one destination change per trip; a second is refused.
func TestSecondDestinationChangeIsRefused(t *testing.T) {
	s, _, _ := newSvcFull(cfg())
	id := inProgress(t, s)
	_ = s.ChangeDestination(id, quote(55000))
	_ = s.ConfirmPrice(id)

	err := s.ChangeDestination(id, quote(60000))

	if !errors.Is(err, ride.ErrChangeLimit) {
		t.Fatalf("err = %v, want ErrChangeLimit", err)
	}
}
