package tests

import (
	"errors"
	"testing"
	"time"

	"cocoyashi/ride"
)

func offeredTo(t *testing.T, s *ride.Service, id ride.RideID) ride.DriverID {
	t.Helper()
	d, ok := s.OfferedTo(id)
	if !ok {
		t.Fatal("no open offer")
	}
	return d
}

// MATCH-2, required: an unanswered offer closes after the accept time and goes to the next driver.
func TestUnansweredOfferPassesToNextDriver(t *testing.T) {
	s, clk := newSvc()
	id, _ := s.RequestRide("rider-1", quote(41000))
	s.Dispatch(id, near("d1", "d2"))

	clk.Advance(15 * time.Second)

	if got := offeredTo(t, s, id); got != "d2" {
		t.Fatalf("offered to %q, want d2", got)
	}
}

// MATCH-2, forbidden: the first driver taps accept after the offer timed out and went to another driver.
func TestAcceptAfterTimeoutIsRefused(t *testing.T) {
	s, clk := newSvc()
	id, _ := s.RequestRide("rider-1", quote(41000))
	s.Dispatch(id, near("d1", "d2"))
	clk.Advance(16 * time.Second)

	err := s.Accept(id, "d1")

	if !errors.Is(err, ride.ErrOfferClosed) {
		t.Fatalf("err = %v, want ErrOfferClosed", err)
	}
	if s.State(id) == ride.Assigned {
		t.Fatal("the late driver was assigned the ride")
	}
	if got := offeredTo(t, s, id); got != "d2" {
		t.Fatalf("offer is with %q, want d2", got)
	}
}

// MATCH-1, required: when the first driver declines, the offer goes to the next, one driver at a time.
func TestDeclinedOfferGoesToNextDriver(t *testing.T) {
	s, _ := newSvc()
	id, _ := s.RequestRide("rider-1", quote(41000))
	s.Dispatch(id, near("d1", "d2", "d3"))
	if got := offeredTo(t, s, id); got != "d1" {
		t.Fatalf("first offer with %q, want d1", got)
	}

	s.Decline(id, "d1")

	if got := offeredTo(t, s, id); got != "d2" {
		t.Fatalf("offered to %q, want d2", got)
	}
}

// MATCH-1, forbidden: two drivers both try to accept; only one is assigned.
func TestOnlyOneDriverIsAssigned(t *testing.T) {
	s, _ := newSvc()
	id, _ := s.RequestRide("rider-1", quote(41000))
	s.Dispatch(id, near("d1", "d2"))

	err1 := s.Accept(id, "d1")
	err2 := s.Accept(id, "d2")

	if err1 != nil {
		t.Fatalf("d1: %v", err1)
	}
	if !errors.Is(err2, ride.ErrOfferClosed) {
		t.Fatalf("d2 err = %v, want ErrOfferClosed", err2)
	}
	if got := s.Driver(id); got != "d1" {
		t.Fatalf("assigned to %q, want d1", got)
	}
}

// MATCH-4, required: every driver passes, so the request ends and no driver is reported found.
func TestRequestEndsWhenEveryDriverTimesOut(t *testing.T) {
	s, clk := newSvc()
	id, _ := s.RequestRide("rider-1", quote(41000))
	s.Dispatch(id, near("d1", "d2", "d3"))

	clk.Advance(45 * time.Second)

	if got := s.State(id); got != ride.NoDriver {
		t.Fatalf("state = %v, want NoDriver", got)
	}
}

// MATCH-4, required: after no driver was found the rider can request again (REQ-2 does not block).
func TestRiderCanRequestAgainAfterNoDriver(t *testing.T) {
	s, clk := newSvc()
	id, _ := s.RequestRide("rider-1", quote(41000))
	s.Dispatch(id, near("d1"))
	clk.Advance(15 * time.Second)
	if s.State(id) != ride.NoDriver {
		t.Fatal("setup: request should have ended")
	}

	if _, err := s.RequestRide("rider-1", quote(41000)); err != nil {
		t.Fatalf("request again: %v", err)
	}
}

// MATCH-4, forbidden: no driver is available at all, so the request does not stay open.
func TestRequestEndsAtOnceWhenNoDriverIsNearby(t *testing.T) {
	s, _ := newSvc()
	id, _ := s.RequestRide("rider-1", quote(41000))

	s.Dispatch(id, nil)

	if got := s.State(id); got != ride.NoDriver {
		t.Fatalf("state = %v, want NoDriver", got)
	}
}

// MATCH-5, required: the search stops at the maximum search time even with drivers left to try.
func TestSearchEndsAtMaxSearchTime(t *testing.T) {
	s, clk := newSvc() // 15 s per offer, 60 s maximum search
	id, _ := s.RequestRide("rider-1", quote(41000))
	s.Dispatch(id, near("d1", "d2", "d3", "d4", "d5", "d6"))

	clk.Advance(60 * time.Second)

	if got := s.State(id); got != ride.NoDriver {
		t.Fatalf("state = %v, want NoDriver at the 60 s limit with 2 drivers still untried", got)
	}
}

// MATCH-5, forbidden: a driver beyond the search radius is never offered the ride.
func TestDriverOutsideRadiusIsNotOffered(t *testing.T) {
	s, clk := newSvc() // radius 5 km
	id, _ := s.RequestRide("rider-1", quote(41000))
	s.Dispatch(id, []ride.Candidate{{ID: "far", DistanceKm: 12}, {ID: "close", DistanceKm: 2}})

	if got := offeredTo(t, s, id); got != "close" {
		t.Fatalf("offered to %q, want close", got)
	}
	clk.Advance(15 * time.Second)
	if got := s.State(id); got != ride.NoDriver {
		t.Fatalf("state = %v, want NoDriver: the far driver must not get the next offer", got)
	}
}
