package tests

import (
	"errors"
	"testing"
	"time"

	"cocoyashi/ride"
)

// arrived takes a ride to the Arrived state: the driver is at the pickup.
func arrived(t *testing.T, s *ride.Service) ride.RideID {
	t.Helper()
	id := assigned(t, s)
	if err := s.Arrive(id, "d1"); err != nil {
		t.Fatal(err)
	}
	return id
}

// CANCEL-3, required: after the wait time the driver can cancel as a no-show; the rider pays the fee, all to the driver.
func TestNoShowAfterWaitChargesFeeToDriver(t *testing.T) {
	s, clk := newSvc()
	id := arrived(t, s)
	clk.Advance(5 * time.Minute)

	if err := s.NoShowCancel(id, "d1"); err != nil {
		t.Fatalf("no-show cancel: %v", err)
	}
	if got := s.FeeCharged(id); got != 3000 {
		t.Fatalf("fee = %d, want 3000", got)
	}
	if got := s.DriverCredit("d1"); got != 3000 {
		t.Fatalf("driver credit = %d, want 3000", got)
	}
}

// CANCEL-3, forbidden: before the wait time has passed the rider is not charged.
func TestEarlyNoShowIsRefused(t *testing.T) {
	s, clk := newSvc()
	id := arrived(t, s)
	clk.Advance(4*time.Minute + 59*time.Second)

	err := s.NoShowCancel(id, "d1")

	if !errors.Is(err, ride.ErrTooEarly) {
		t.Fatalf("err = %v, want ErrTooEarly", err)
	}
	if got := s.FeeCharged(id); got != 0 {
		t.Fatalf("fee = %d charged for an early no-show", got)
	}
	if got := s.State(id); got != ride.Arrived {
		t.Fatalf("state = %v, want Arrived", got)
	}
}
