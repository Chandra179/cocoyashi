package tests

import (
	"testing"

	"cocoyashi/ride"
)

// no caps: demand multiples are in hundredths, so 150 is 1.5x.
var noCaps = ride.SurgeCaps{}

// SURGE-1, required: when demand is high the higher total and a demand notice come with the upfront estimate.
func TestSurgeIsShownInTheUpfrontFare(t *testing.T) {
	q := ride.EstimateSurge(exampleRates(), 10_000, 20, 150, noCaps) // normal fare 41000

	if q.Fare != 61500 {
		t.Fatalf("fare = %d, want 61500 (41000 x 1.5)", q.Fare)
	}
	if !q.Surged {
		t.Fatal("quote carries no demand notice")
	}
}

// SURGE-1: with no extra demand there is no surge and no notice.
func TestNoSurgeWhenDemandIsNormal(t *testing.T) {
	q := ride.EstimateSurge(exampleRates(), 10_000, 20, 100, noCaps)

	if q.Fare != 41000 || q.Surged {
		t.Fatalf("fare = %d, surged = %v; want 41000 and no notice", q.Fare, q.Surged)
	}
}

// SURGE-1: the surge never exceeds the platform cap.
func TestSurgeIsCappedAtThePlatformCap(t *testing.T) {
	caps := ride.SurgeCaps{Platform: 200}

	q := ride.EstimateSurge(exampleRates(), 10_000, 20, 300, caps)

	if q.Fare != 82000 {
		t.Fatalf("fare = %d, want 82000 (41000 x 2.0 cap, not x 3.0)", q.Fare)
	}
}

// SURGE-1: when a legal cap is lower than the platform cap, the legal cap wins.
func TestLowerLegalCapWins(t *testing.T) {
	caps := ride.SurgeCaps{Platform: 200, Legal: 150}

	q := ride.EstimateSurge(exampleRates(), 10_000, 20, 300, caps)

	if q.Fare != 61500 {
		t.Fatalf("fare = %d, want 61500 (41000 x 1.5 legal cap)", q.Fare)
	}
}

// SURGE-1: a legal cap higher than the platform cap does not lift the platform cap.
func TestHigherLegalCapDoesNotRaiseThePlatformCap(t *testing.T) {
	caps := ride.SurgeCaps{Platform: 200, Legal: 300}

	q := ride.EstimateSurge(exampleRates(), 10_000, 20, 300, caps)

	if q.Fare != 82000 {
		t.Fatalf("fare = %d, want 82000", q.Fare)
	}
}

// SURGE-1, forbidden: demand rising after the rider confirmed does not raise what is charged.
func TestSurgeAfterConfirmingDoesNotRaiseTheFare(t *testing.T) {
	s, _, gw := newSvcFull(cfg())
	confirmed := ride.EstimateSurge(exampleRates(), 10_000, 20, 100, noCaps) // 41000 at normal demand
	id, _ := s.RequestRide("rider-1", confirmed)
	s.Dispatch(id, near("d1"))
	_ = s.Accept(id, "d1")
	_ = s.Arrive(id, "d1")
	_ = s.Start(id, "d1")
	_ = ride.EstimateSurge(exampleRates(), 10_000, 20, 250, noCaps) // demand rises mid-trip

	_ = s.Complete(id)

	if got := gw.captured[string(id)]; got != 41000 {
		t.Fatalf("charged %d, want the confirmed 41000", got)
	}
}
