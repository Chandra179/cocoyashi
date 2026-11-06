package tests

import (
	"testing"
	"time"

	"cocoyashi/ride"
)

// DCANCEL-1, required: the driver cancels after accepting; the request goes to the next driver.
func TestDriverCancelRedispatchesToNextDriver(t *testing.T) {
	s, _ := newSvc()
	id, _ := s.RequestRide("rider-1", quote(41000))
	s.Dispatch(id, near("d1", "d2"))
	if err := s.Accept(id, "d1"); err != nil {
		t.Fatal(err)
	}

	if err := s.DriverCancel(id, "d1"); err != nil {
		t.Fatalf("driver cancel: %v", err)
	}

	if got := s.State(id); got != ride.Offering {
		t.Fatalf("state = %v, want Offering", got)
	}
	if got := offeredTo(t, s, id); got != "d2" {
		t.Fatalf("offered to %q, want d2 (and not d1 again)", got)
	}
}

// DCANCEL-1, required: the rider is told the driver cancelled.
func TestRiderIsToldWhenDriverCancels(t *testing.T) {
	s, _ := newSvc()
	id := assigned(t, s)

	_ = s.DriverCancel(id, "d1")

	got := s.Notices("rider-1")
	if len(got) != 1 || got[0] != ride.NoticeDriverCancelled {
		t.Fatalf("notices = %v, want [NoticeDriverCancelled]", got)
	}
}

// DCANCEL-1, forbidden: the rider is charged nothing for a driver cancel, even after the grace period.
func TestRiderPaysNothingForDriverCancel(t *testing.T) {
	s, clk := newSvc()
	id := assigned(t, s)
	clk.Advance(10 * time.Minute)

	_ = s.DriverCancel(id, "d1")

	if got := s.FeeCharged(id); got != 0 {
		t.Fatalf("fee = %d, want 0", got)
	}
	if got := s.DriverCredit("d1"); got != 0 {
		t.Fatalf("driver credit = %d, want 0", got)
	}
}
