package tests

import (
	"testing"
	"time"
)

// EARN-1, required: the weekly payout is the fares less the market's commission rate (Indonesia: 8%), paid once.
func TestWeeklyPayoutIsFaresLessMarketCommission(t *testing.T) {
	c := cfg()
	c.CommissionBPS = 800
	s, _, _ := newSvcFull(c)
	id := inProgress(t, s)
	_ = s.Complete(id) // fare 41000

	paid := s.RunPayout()

	if paid["d1"] != 37720 {
		t.Fatalf("payout = %d, want 37720 (41000 less 8%%)", paid["d1"])
	}
	if again := s.RunPayout(); again["d1"] != 0 {
		t.Fatalf("second run paid %d, want 0", again["d1"])
	}
}

// EARN-1, forbidden: commission above the legal cap is never taken, whatever rate is configured.
func TestCommissionNeverExceedsLegalCap(t *testing.T) {
	c := cfg()
	c.CommissionBPS = 2000
	c.CommissionCapBPS = 800
	s, _, _ := newSvcFull(c)
	id := inProgress(t, s)
	_ = s.Complete(id)

	if got := s.RunPayout()["d1"]; got != 37720 {
		t.Fatalf("payout = %d, want 37720: commission capped at 8%%", got)
	}
}

// EARN-1: cancellation and no-show fees carry no commission.
func TestCancelFeeIsPaidWithoutCommission(t *testing.T) {
	s, clk, _ := newSvcFull(cfg())
	id := assigned(t, s)
	clk.Advance(6 * time.Minute)
	_ = s.Cancel(id, 3000)

	if got := s.RunPayout()["d1"]; got != 3000 {
		t.Fatalf("payout = %d, want 3000", got)
	}
}
