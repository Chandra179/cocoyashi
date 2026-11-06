package tests

import (
	"testing"

	"cocoyashi/ride"
)

// PAY-3, required: the hold is the fare at the surge cap in force, plus known tolls.
func TestHoldCoversSurgeCapAndKnownTolls(t *testing.T) {
	s, _, gw := newSvcFull(cfg())
	q := ride.EstimateSurge(exampleRates(), 10_000, 20, 150, ride.SurgeCaps{Platform: 200}) // 61500 with surge
	q.Tolls = 6000

	id, _ := s.RequestRide("rider-1", q)

	if got := gw.holds[string(id)]; got != 88000 {
		t.Fatalf("hold = %d, want 88000 (41000 x 2.0 cap + 6000 tolls)", got)
	}
}

// PAY-3, forbidden: a rider detour that pushes the fare past the hold is charged in one transaction, not two.
func TestOverageBeyondTheHoldIsChargedOnce(t *testing.T) {
	s, _, gw := newSvcFull(cfg())
	q := ride.EstimateSurge(exampleRates(), 10_000, 20, 100, ride.SurgeCaps{Platform: 200}) // hold 82000
	id, _ := s.RequestRide("rider-1", q)
	s.Dispatch(id, near("d1"))
	_ = s.Accept(id, "d1")
	_ = s.Arrive(id, "d1")
	_ = s.Start(id, "d1")

	_ = s.CompleteWith(id, ride.Facts{Cause: ride.RiderDetour, EstMinutes: 30, ActualMinutes: 90, Metered: 100000})

	if gw.captureCall != 1 || gw.captured[string(id)] != 100000 {
		t.Fatalf("capture calls = %d, captured = %d; want one capture of 100000", gw.captureCall, gw.captured[string(id)])
	}
}

// PAY-3: a confirmed price change moves the hold to the new fare.
func TestConfirmedPriceChangeUpdatesTheHold(t *testing.T) {
	s, _, gw := newSvcFull(cfg())
	id := inProgress(t, s)
	_ = s.ChangeDestination(id, quote(55000))
	_ = s.ConfirmPrice(id)

	if got := gw.holds[string(id)]; got != 55000 {
		t.Fatalf("hold = %d, want 55000", got)
	}
}
