package tests

import (
	"fmt"
	"testing"
	"time"

	"cocoyashi/ride"
)

// ignoreOne offers a fresh ride to d1 alone and lets the offer time out.
func ignoreOne(t *testing.T, s *ride.Service, clk *fakeClock, rider ride.RiderID) {
	t.Helper()
	id, err := s.RequestRide(rider, quote(41000))
	if err != nil {
		t.Fatal(err)
	}
	s.Dispatch(id, near("d1"))
	clk.Advance(15 * time.Second)
	_ = s.State(id) // lets the timeout be evaluated
}

// MATCH-3, required: three ignored offers in a row take the driver offline.
func TestThreeIgnoredOffersTakeTheDriverOffline(t *testing.T) {
	s, clk := newSvc()
	for i, rider := range []ride.RiderID{"r1", "r2", "r3"} {
		if !s.IsOnline("d1") {
			t.Fatalf("d1 offline after only %d ignored offers", i)
		}
		ignoreOne(t, s, clk, rider)
	}

	if s.IsOnline("d1") {
		t.Fatal("d1 still online after 3 ignored offers")
	}
}

// MATCH-3: an offline driver gets no new offers, and is back after the timeout.
func TestOfflineDriverGetsNoOffersUntilTimeoutEnds(t *testing.T) {
	s, clk := newSvc()
	for _, rider := range []ride.RiderID{"r1", "r2", "r3"} {
		ignoreOne(t, s, clk, rider)
	}
	id, _ := s.RequestRide("r4", quote(41000))
	s.Dispatch(id, near("d1"))
	if got := s.State(id); got != ride.NoDriver {
		t.Fatalf("state = %v: an offline driver was offered the ride", got)
	}

	clk.Advance(5 * time.Minute)

	if !s.IsOnline("d1") {
		t.Fatal("d1 still offline after the 5 minute timeout")
	}
}

// MATCH-3: only offers ignored in a row count; accepting one resets the count.
func TestAcceptingAnOfferResetsTheStreak(t *testing.T) {
	s, clk := newSvc()
	ignoreOne(t, s, clk, "r1")
	ignoreOne(t, s, clk, "r2")
	id, _ := s.RequestRide("r3", quote(41000))
	s.Dispatch(id, near("d1"))
	if err := s.Accept(id, "d1"); err != nil {
		t.Fatal(err)
	}

	ignoreOne(t, s, clk, "r4")
	ignoreOne(t, s, clk, "r5")

	if !s.IsOnline("d1") {
		t.Fatal("d1 went offline although they accepted an offer between the ignored ones")
	}
}

// MATCH-3: a decline counts like an ignored offer.
func TestDeclinesCountTowardsTheStreak(t *testing.T) {
	s, _ := newSvc()
	for _, rider := range []ride.RiderID{"r1", "r2", "r3"} {
		id, _ := s.RequestRide(rider, quote(41000))
		s.Dispatch(id, near("d1"))
		s.Decline(id, "d1")
	}

	if s.IsOnline("d1") {
		t.Fatal("d1 still online after 3 declines in a row")
	}
}

// MATCH-3, forbidden: an offer that closed because the rider cancelled does not count as ignored.
func TestOffersClosedByRiderCancelAreNotCounted(t *testing.T) {
	s, _ := newSvc()
	for i := 0; i < 5; i++ {
		rider := ride.RiderID(fmt.Sprintf("r%d", i))
		id, _ := s.RequestRide(rider, quote(41000))
		s.Dispatch(id, near("d1"))
		_ = s.Cancel(id, 0)
	}

	if !s.IsOnline("d1") {
		t.Fatal("d1 was taken offline for offers the riders cancelled")
	}
}
