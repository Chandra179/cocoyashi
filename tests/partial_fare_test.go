package tests

import (
	"testing"

	"cocoyashi/ride"
)

// partial ends a trip with the given upfront fare short of the destination and returns what was charged.
func partial(t *testing.T, upfront ride.Money, doneM, totalM int64) (ride.Money, *ride.Service, ride.RideID) {
	t.Helper()
	s, _, gw := newSvcFull(cfg()) // minimum fare 12000
	id, _ := s.RequestRide("rider-1", quote(upfront))
	s.Dispatch(id, near("d1"))
	_ = s.Accept(id, "d1")
	_ = s.Arrive(id, "d1")
	_ = s.Start(id, "d1")
	if err := s.CompleteWith(id, ride.Facts{EndedShort: true, CompletedMeters: doneM, TotalMeters: totalM}); err != nil {
		t.Fatalf("complete: %v", err)
	}
	return gw.captured[string(id)], s, id
}

// TRIP-2, required: half the distance, half the upfront fare.
func TestPartialTripIsChargedProRata(t *testing.T) {
	if got, _, _ := partial(t, 40000, 5000, 10000); got != 20000 {
		t.Fatalf("charged %d, want 20000", got)
	}
}

// TRIP-2, required: a very short partial trip is raised to the minimum fare.
func TestPartialTripNeverBelowMinimumFare(t *testing.T) {
	if got, _, _ := partial(t, 40000, 1000, 10000); got != 12000 {
		t.Fatalf("charged %d, want 12000", got)
	}
}

// TRIP-2, forbidden: a partial trip is never charged more than the upfront fare, even over the minimum.
func TestPartialTripNeverAboveUpfrontFare(t *testing.T) {
	if got, _, _ := partial(t, 10000, 9000, 10000); got > 10000 {
		t.Fatalf("charged %d, above the upfront fare 10000", got)
	}
}

// TRIP-1, required: the rider is told the trip ended short.
func TestRiderIsToldWhenTripEndsShort(t *testing.T) {
	_, s, _ := partial(t, 40000, 5000, 10000)

	got := s.Notices("rider-1")
	if len(got) != 1 || got[0] != ride.NoticeTripEndedShort {
		t.Fatalf("notices = %v, want [NoticeTripEndedShort]", got)
	}
}
