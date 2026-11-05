package tests

import (
	"errors"
	"testing"
	"time"

	"cocoyashi/ride"
)

// CANCEL-2: late cancel costs the flat fee, shown first, 100% to the driver.
func TestLateCancelChargesFlatFeeToDriver(t *testing.T) {
	s, clk, _ := newSvc()
	id := assigned(t, s, "r1", 25000, ride.Card)
	clk.Advance(6 * time.Minute)

	fee := s.CancelFee(id)
	if fee != 3000 {
		t.Fatalf("fee shown = %d, want 3000", fee)
	}
	charged, err := s.RiderCancel(id, fee)
	if err != nil || charged != 3000 {
		t.Fatalf("charged=%d err=%v", charged, err)
	}
	if got := s.DriverCredit("d1"); got != 3000 {
		t.Fatalf("driver credit = %d, want the whole fee, no commission", got)
	}
}

func TestCancelWithoutSeeingFeeIsRefused(t *testing.T) {
	s, clk, _ := newSvc()
	id := assigned(t, s, "r1", 25000, ride.Card)
	clk.Advance(6 * time.Minute)

	if _, err := s.RiderCancel(id, 0); !errors.Is(err, ride.ErrFeeNotAcknowledged) {
		t.Fatalf("want ErrFeeNotAcknowledged, got %v", err)
	}
	if s.State(id) == ride.Cancelled {
		t.Fatal("ride cancelled without the rider confirming the fee")
	}
	if s.DriverCredit("d1") != 0 {
		t.Fatal("driver credited for a cancel that did not happen")
	}
}
