package tests

import (
	"testing"

	"cocoyashi/ride"
)

// PAY-1: the final fare is charged once, even if settlement is retried.
func TestChargedOnceAtTripEnd(t *testing.T) {
	s, _, gw := newSvc()
	id := inProgress(t, s, "r1", 25000, ride.Card)
	st, err := s.Settle(id, ride.Facts{})
	if err != nil || !st.Charged || st.Fare != 25000 {
		t.Fatalf("settlement=%+v err=%v", st, err)
	}
	if gw.paid[string(id)] != 25000 {
		t.Fatalf("gateway paid %v", gw.paid)
	}
}

func TestRetryDoesNotChargeTwice(t *testing.T) {
	s, _, gw := newSvc()
	id := inProgress(t, s, "r1", 25000, ride.Card)
	s.Settle(id, ride.Facts{})
	s.Settle(id, ride.Facts{})
	if gw.calls != 1 {
		t.Fatalf("gateway called %d times, want 1", gw.calls)
	}
}

// PAYM-1: a cash trip places no card charge.
func TestCashTripNotCharged(t *testing.T) {
	s, _, gw := newSvc()
	id := inProgress(t, s, "r1", 25000, ride.Cash)
	st, err := s.Settle(id, ride.Facts{})
	if err != nil || st.Charged {
		t.Fatalf("settlement=%+v err=%v", st, err)
	}
	if gw.calls != 0 {
		t.Fatalf("gateway called %d times for cash", gw.calls)
	}
}
