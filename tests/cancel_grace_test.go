package tests

import (
	"testing"
	"time"

	"cocoyashi/ride"
)

// CANCEL-1: no fee before a driver is assigned, or inside the grace window.
func TestFreeCancelBeforeAssignment(t *testing.T) {
	s, _, _ := newSvc()
	id := mustRequest(t, s, "r1", 25000, ride.Card)
	fee, err := s.RiderCancel(id, 0)
	if err != nil || fee != 0 {
		t.Fatalf("fee=%d err=%v, want free cancel", fee, err)
	}
	if s.State(id) != ride.Cancelled {
		t.Fatalf("state = %v", s.State(id))
	}
}

func TestFreeCancelInsideGrace(t *testing.T) {
	s, clk, _ := newSvc()
	id := assigned(t, s, "r1", 25000, ride.Card)
	clk.Advance(5 * time.Minute) // exactly the grace period: still free
	if got := s.CancelFee(id); got != 0 {
		t.Fatalf("fee inside window = %d, want 0", got)
	}
	fee, err := s.RiderCancel(id, 0)
	if err != nil || fee != 0 {
		t.Fatalf("fee=%d err=%v", fee, err)
	}
}

func TestGraceComesFromConfig(t *testing.T) {
	c := cfg()
	c.CancelGrace = 2 * time.Minute
	clk := &fakeClock{}
	s := ride.NewService(c, clk, &fakeGateway{})
	id, _ := s.RequestRide("r1", quote(25000), ride.Card)
	s.Dispatch(id, []ride.DriverID{"d1"})
	s.Accept(id, "d1")
	clk.Advance(3 * time.Minute)
	if got := s.CancelFee(id); got != c.CancelFee {
		t.Fatalf("fee = %d, want %d (grace is 2 minutes in this market)", got, c.CancelFee)
	}
}
