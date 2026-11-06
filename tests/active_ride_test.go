package tests

import (
	"errors"
	"testing"

	"cocoyashi/ride"
)

// REQ-2, forbidden: a second request while a ride is active is rejected and the first ride is unchanged.
func TestSecondRequestRejectedWhileRideActive(t *testing.T) {
	s, _ := newSvc()
	first, err := s.RequestRide("rider-1", quote(41000))
	if err != nil {
		t.Fatalf("first request: %v", err)
	}

	id, err := s.RequestRide("rider-1", quote(20000))

	if !errors.Is(err, ride.ErrActiveRide) {
		t.Fatalf("err = %v, want ErrActiveRide", err)
	}
	if id != "" {
		t.Fatalf("second ride %q was created", id)
	}
	if got := s.Upfront(first); got != 41000 {
		t.Fatalf("existing ride changed: upfront = %d", got)
	}
}

// REQ-2: a different rider is not blocked by someone else's ride.
func TestOtherRiderCanStillRequest(t *testing.T) {
	s, _ := newSvc()
	if _, err := s.RequestRide("rider-1", quote(41000)); err != nil {
		t.Fatal(err)
	}
	if _, err := s.RequestRide("rider-2", quote(20000)); err != nil {
		t.Fatalf("rider-2 blocked: %v", err)
	}
}
