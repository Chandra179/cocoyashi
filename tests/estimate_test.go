package tests

import (
	"errors"
	"testing"

	"cocoyashi/ride"
)

// REQ-1: the fare estimate is shown before the request is confirmed.
func TestRequestWithEstimateIsCreated(t *testing.T) {
	s, _, _ := newSvc()
	id, err := s.RequestRide("r1", quote(25000), ride.Card)
	if err != nil {
		t.Fatalf("want request created, got %v", err)
	}
	if got := s.Upfront(id); got != 25000 {
		t.Fatalf("upfront fare = %d, want 25000", got)
	}
}

func TestRequestBlockedWithoutEstimate(t *testing.T) {
	s, _, _ := newSvc()
	_, err := s.RequestRide("r1", ride.Quote{Available: false}, ride.Card)
	if !errors.Is(err, ride.ErrNoEstimate) {
		t.Fatalf("want ErrNoEstimate, got %v", err)
	}
	// nothing was created, so a later valid request is not an "active ride" conflict
	if _, err := s.RequestRide("r1", quote(25000), ride.Card); err != nil {
		t.Fatalf("later request: %v", err)
	}
}
