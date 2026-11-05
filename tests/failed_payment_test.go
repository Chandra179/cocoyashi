package tests

import (
	"errors"
	"testing"

	"cocoyashi/ride"
)

// PAY-2: a failed payment leaves a balance that blocks new rides; the driver is still paid.
func TestFailedPaymentBlocksNextRequest(t *testing.T) {
	s, _, gw := newSvc()
	gw.fail = true
	id := inProgress(t, s, "r1", 100000, ride.Card)
	st, err := s.Settle(id, ride.Facts{})
	if err != nil || st.Charged {
		t.Fatalf("settlement=%+v err=%v", st, err)
	}
	if s.Owed("r1") != 100000 {
		t.Fatalf("owed = %d", s.Owed("r1"))
	}
	if _, err := s.RequestRide("r1", quote(25000), ride.Card); !errors.Is(err, ride.ErrOutstandingBalance) {
		t.Fatalf("want ErrOutstandingBalance, got %v", err)
	}
}

func TestDriverPaidWhenRiderPaymentFails(t *testing.T) {
	s, _, gw := newSvc()
	gw.fail = true
	id := inProgress(t, s, "r1", 100000, ride.Card)
	st, _ := s.Settle(id, ride.Facts{})
	if st.DriverEarning != 80000 || s.DriverCredit("d1") != 80000 {
		t.Fatalf("earning=%d credit=%d, want 80000 (fare minus 20%%)", st.DriverEarning, s.DriverCredit("d1"))
	}
}

func TestPayingBalanceUnblocksRider(t *testing.T) {
	s, _, gw := newSvc()
	gw.fail = true
	id := inProgress(t, s, "r1", 100000, ride.Card)
	s.Settle(id, ride.Facts{})
	gw.fail = false
	if err := s.PayOutstanding("r1"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.RequestRide("r1", quote(25000), ride.Card); err != nil {
		t.Fatalf("request after paying: %v", err)
	}
}
