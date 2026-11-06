package tests

import (
	"errors"

	"cocoyashi/ride"
	"testing"
	"time"
)

// CANCEL-2, required: after the free window the flat fee is charged and credited in full to the driver.
func TestLateCancelChargesFlatFeeToDriver(t *testing.T) {
	s, clk := newSvc()
	id := assigned(t, s)
	clk.Advance(5*time.Minute + time.Second)

	if got := s.CancelQuote(id); got != 3000 {
		t.Fatalf("fee shown = %d, want 3000", got)
	}
	if err := s.Cancel(id, 3000); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	if got := s.FeeCharged(id); got != 3000 {
		t.Fatalf("fee charged = %d, want 3000", got)
	}
	if got := s.DriverCredit("d1"); got != 3000 {
		t.Fatalf("driver credit = %d, want 3000 (100%%, no commission)", got)
	}
}

// CANCEL-2, forbidden: the cancel does not complete unless the rider confirmed the fee shown.
func TestCancelWithUnconfirmedFeeIsRefused(t *testing.T) {
	s, clk := newSvc()
	id := assigned(t, s)
	clk.Advance(6 * time.Minute)

	err := s.Cancel(id, 0)

	if !errors.Is(err, ride.ErrFeeNotAcknowledged) {
		t.Fatalf("err = %v, want ErrFeeNotAcknowledged", err)
	}
	if got := s.State(id); got != ride.Assigned {
		t.Fatalf("state = %v, want the ride still Assigned", got)
	}
	if got := s.FeeCharged(id); got != 0 {
		t.Fatalf("fee = %d charged without confirmation", got)
	}
}

// CANCEL-2, required: the fee also applies when the driver has already arrived at the pickup.
func TestLateCancelAfterDriverArrivedChargesFee(t *testing.T) {
	s, clk := newSvc()
	id := arrived(t, s)
	clk.Advance(6 * time.Minute)

	if err := s.Cancel(id, 3000); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	if got := s.FeeCharged(id); got != 3000 {
		t.Fatalf("fee = %d, want 3000", got)
	}
}
