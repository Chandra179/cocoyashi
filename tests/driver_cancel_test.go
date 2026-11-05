package tests

import (
	"testing"

	"cocoyashi/ride"
)

// DCANCEL-1: a driver cancel costs the rider nothing and the ride goes to the next driver.
func TestDriverCancelRedispatches(t *testing.T) {
	s, _, _ := newSvc()
	id := assigned(t, s, "r1", 25000, ride.Card)

	if err := s.DriverCancel(id); err != nil {
		t.Fatal(err)
	}
	if s.Owed("r1") != 0 {
		t.Fatal("rider charged for a driver cancel")
	}
	d, ok := s.CurrentOffer(id)
	if !ok || d != "d2" {
		t.Fatalf("offer = %v %v, want d2 (d1 must not get it again)", d, ok)
	}
}

func TestDriverCancelWithNobodyLeftEndsFree(t *testing.T) {
	s, _, _ := newSvc()
	id := mustRequest(t, s, "r1", 25000, ride.Card)
	s.Dispatch(id, []ride.DriverID{"d1"})
	s.Accept(id, "d1")
	s.DriverCancel(id)
	if s.State(id) != ride.NoDriver || s.Owed("r1") != 0 {
		t.Fatalf("state=%v owed=%d", s.State(id), s.Owed("r1"))
	}
}
