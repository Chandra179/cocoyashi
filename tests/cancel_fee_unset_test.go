package tests

import (
	"testing"
	"time"
)

// CANCEL-4, required and forbidden: where cancel_fee_amount is unset, a late cancel and a no-show charge nothing.
func TestUnsetFeeChargesNothing(t *testing.T) {
	c := cfg()
	c.CancelFee = nil

	t.Run("late rider cancel", func(t *testing.T) {
		s, clk := newSvcWith(c)
		id := assigned(t, s)
		clk.Advance(6 * time.Minute)

		if err := s.Cancel(id, 0); err != nil {
			t.Fatalf("cancel blocked: %v", err)
		}
		if got := s.FeeCharged(id); got != 0 {
			t.Fatalf("fee = %d, want 0", got)
		}
	})
	t.Run("driver no-show", func(t *testing.T) {
		s, clk := newSvcWith(c)
		id := arrived(t, s)
		clk.Advance(6 * time.Minute)

		if err := s.NoShowCancel(id, "d1"); err != nil {
			t.Fatalf("no-show blocked: %v", err)
		}
		if got := s.FeeCharged(id); got != 0 || s.DriverCredit("d1") != 0 {
			t.Fatalf("fee = %d, credit = %d, want 0", got, s.DriverCredit("d1"))
		}
	})
	t.Run("rider can request again", func(t *testing.T) {
		s, clk := newSvcWith(c)
		id := assigned(t, s)
		clk.Advance(6 * time.Minute)
		_ = s.Cancel(id, 0)

		if _, err := s.RequestRide("rider-1", quote(41000)); err != nil {
			t.Fatalf("next request blocked: %v", err)
		}
	})
}
