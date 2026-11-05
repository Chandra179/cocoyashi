package tests

import (
	"errors"
	"testing"
	"time"

	"cocoyashi/ride"
)

// CANCEL-3: no-show cancel only after the wait time, same fee, 100% to the driver.
func TestNoShowBeforeWaitRefused(t *testing.T) {
	s, clk, _ := newSvc()
	id := assigned(t, s, "r1", 25000, ride.Card)
	s.Arrive(id)
	clk.Advance(4 * time.Minute)

	if _, err := s.DriverNoShow(id); !errors.Is(err, ride.ErrTooEarly) {
		t.Fatalf("want ErrTooEarly, got %v", err)
	}
	if s.Owed("r1") != 0 {
		t.Fatal("rider charged for an early no-show")
	}
}

func TestNoShowAfterWaitChargesFee(t *testing.T) {
	s, clk, _ := newSvc()
	id := assigned(t, s, "r1", 25000, ride.Card)
	s.Arrive(id)
	clk.Advance(5 * time.Minute)

	fee, err := s.DriverNoShow(id)
	if err != nil || fee != 3000 {
		t.Fatalf("fee=%d err=%v, want 3000", fee, err)
	}
	if got := s.DriverCredit("d1"); got != 3000 {
		t.Fatalf("driver credit = %d, want 3000", got)
	}
	if s.State(id) != ride.Cancelled {
		t.Fatalf("state = %v", s.State(id))
	}
}
