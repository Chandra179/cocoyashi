package tests

import (
	"errors"
	"testing"
	"time"

	"cocoyashi/ride"
)

func dispatched(t *testing.T) (*ride.Service, *fakeClock, ride.RideID) {
	t.Helper()
	s, clk, _ := newSvc()
	id := mustRequest(t, s, "r1", 25000, ride.Card)
	if err := s.Dispatch(id, []ride.DriverID{"d1", "d2", "d3"}); err != nil {
		t.Fatal(err)
	}
	return s, clk, id
}

// MATCH-1: one open offer per ride request.
func TestOnlyOneOpenOffer(t *testing.T) {
	s, _, id := dispatched(t)
	d, ok := s.CurrentOffer(id)
	if !ok || d != "d1" {
		t.Fatalf("offer = %v %v, want d1", d, ok)
	}
	if err := s.Accept(id, "d2"); !errors.Is(err, ride.ErrOfferClosed) {
		t.Fatalf("d2 accepted without an offer: %v", err)
	}
}

// MATCH-2: 15 seconds to accept, then the next driver.
func TestOfferPassesAfterTimeout(t *testing.T) {
	s, clk, id := dispatched(t)
	clk.Advance(15 * time.Second)
	if d, _ := s.CurrentOffer(id); d != "d2" {
		t.Fatalf("after timeout offer = %v, want d2", d)
	}
}

func TestOfferStillOpenJustBeforeTimeout(t *testing.T) {
	s, clk, id := dispatched(t)
	clk.Advance(14 * time.Second)
	if d, _ := s.CurrentOffer(id); d != "d1" {
		t.Fatalf("offer = %v, want d1", d)
	}
}

func TestAcceptAfterTimeoutRejected(t *testing.T) {
	s, clk, id := dispatched(t)
	clk.Advance(16 * time.Second)
	if err := s.Accept(id, "d1"); !errors.Is(err, ride.ErrOfferClosed) {
		t.Fatalf("late accept: want ErrOfferClosed, got %v", err)
	}
	if err := s.Accept(id, "d2"); err != nil {
		t.Fatalf("d2 accept: %v", err)
	}
}

func TestDeclineMovesToNextDriver(t *testing.T) {
	s, _, id := dispatched(t)
	s.Decline(id, "d1")
	if d, _ := s.CurrentOffer(id); d != "d2" {
		t.Fatalf("offer = %v, want d2", d)
	}
}

// MATCH-4 (proposed): when every driver passes, the rider is told and the request ends free.
func TestNoDriverFound(t *testing.T) {
	s, _, id := dispatched(t)
	s.Decline(id, "d1")
	s.Decline(id, "d2")
	s.Decline(id, "d3")
	if s.State(id) != ride.NoDriver {
		t.Fatalf("state = %v, want NoDriver", s.State(id))
	}
	if _, err := s.RequestRide("r1", quote(25000), ride.Card); err != nil {
		t.Fatalf("rider should be free to retry: %v", err)
	}
}
