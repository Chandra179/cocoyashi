package tests

import (
	"errors"
	"testing"

	"cocoyashi/ride"
)

func quote(fare ride.Money) ride.Quote { return ride.Quote{Fare: fare, Available: true} }

// REQ-1, required: a request is created with the estimate the rider saw as its upfront fare.
func TestRequestKeepsTheShownEstimateAsUpfrontFare(t *testing.T) {
	s, _ := newSvc()

	id, err := s.RequestRide("rider-1", quote(41000))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if got := s.Upfront(id); got != 41000 {
		t.Fatalf("upfront = %d, want 41000", got)
	}
	if got := s.State(id); got != ride.Requested {
		t.Fatalf("state = %v, want Requested", got)
	}
}

// REQ-1, forbidden: no estimate, no request, and nothing is sent with an unknown fare.
func TestRequestBlockedWithoutEstimate(t *testing.T) {
	s, _ := newSvc()

	id, err := s.RequestRide("rider-1", ride.Quote{})
	if !errors.Is(err, ride.ErrNoEstimate) {
		t.Fatalf("err = %v, want ErrNoEstimate", err)
	}
	if id != "" {
		t.Fatalf("a ride %q was created without an estimate", id)
	}
}
