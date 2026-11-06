package tests

import (
	"testing"
	"time"

	"cocoyashi/ride"
)

// CANCEL-1, required: cancelling before any driver accepted ends the ride and charges nothing.
func TestCancelBeforeAcceptanceIsFree(t *testing.T) {
	s, _ := newSvc()
	id, _ := s.RequestRide("rider-1", quote(41000))
	s.Dispatch(id, near("d1"))

	if err := s.Cancel(id, 0); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	if got := s.State(id); got != ride.Cancelled {
		t.Fatalf("state = %v, want Cancelled", got)
	}
	if got := s.FeeCharged(id); got != 0 {
		t.Fatalf("fee = %d, want 0", got)
	}
}

// CANCEL-1, required and forbidden: inside the free window after assignment no fee is charged.
func TestCancelInsideGracePeriodIsFree(t *testing.T) {
	s, clk := newSvc()
	id := assigned(t, s)
	clk.Advance(4*time.Minute + 59*time.Second)

	if err := s.Cancel(id, 0); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	if got := s.FeeCharged(id); got != 0 {
		t.Fatalf("fee = %d inside the free window, want 0", got)
	}
}
