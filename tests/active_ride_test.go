package tests

import (
	"errors"
	"testing"

	"cocoyashi/ride"
)

// REQ-2: a rider with an active ride cannot request a second one.
func TestSecondRequestRejected(t *testing.T) {
	s, _, _ := newSvc()
	first := mustRequest(t, s, "r1", 25000, ride.Card)
	before := s.State(first)

	_, err := s.RequestRide("r1", quote(30000), ride.Card)
	if !errors.Is(err, ride.ErrActiveRide) {
		t.Fatalf("want ErrActiveRide, got %v", err)
	}
	if s.State(first) != before {
		t.Fatalf("existing ride changed: %v -> %v", before, s.State(first))
	}
}

func TestOtherRiderUnaffected(t *testing.T) {
	s, _, _ := newSvc()
	mustRequest(t, s, "r1", 25000, ride.Card)
	if _, err := s.RequestRide("r2", quote(30000), ride.Card); err != nil {
		t.Fatalf("other rider: %v", err)
	}
}

func TestRequestAllowedAfterRideEnds(t *testing.T) {
	s, _, _ := newSvc()
	id := mustRequest(t, s, "r1", 25000, ride.Card)
	if _, err := s.RiderCancel(id, 0); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	if _, err := s.RequestRide("r1", quote(25000), ride.Card); err != nil {
		t.Fatalf("new request after cancel: %v", err)
	}
}
