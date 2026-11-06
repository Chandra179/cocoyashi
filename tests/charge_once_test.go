package tests

import (
	"testing"

	"cocoyashi/ride"
)

// PAY-1, required: a card request holds the fare on the rider's payment method.
func TestRequestHoldsTheFare(t *testing.T) {
	s, _, gw := newSvcFull(cfg())

	id, _ := s.RequestRide("rider-1", quote(41000))

	if got := gw.holds[string(id)]; got != 41000 {
		t.Fatalf("hold = %d, want 41000", got)
	}
}

func inProgress(t *testing.T, s *ride.Service) ride.RideID {
	t.Helper()
	id := arrived(t, s)
	if err := s.Start(id, "d1"); err != nil {
		t.Fatal(err)
	}
	return id
}

// PAY-1, required: when the trip ends the final fare is charged once.
func TestTripEndChargesTheFareOnce(t *testing.T) {
	s, _, gw := newSvcFull(cfg())
	id := inProgress(t, s)

	if err := s.Complete(id); err != nil {
		t.Fatalf("complete: %v", err)
	}

	if got := gw.captured[string(id)]; got != 41000 {
		t.Fatalf("captured = %d, want 41000", got)
	}
	if got := s.State(id); got != ride.Completed {
		t.Fatalf("state = %v, want Completed", got)
	}
}

// PAY-1, forbidden: a trip that was already charged is not charged a second time on a retry.
func TestRetryDoesNotChargeTwice(t *testing.T) {
	s, _, gw := newSvcFull(cfg())
	id := inProgress(t, s)
	_ = s.Complete(id)

	_ = s.Complete(id)

	if gw.captureCall != 1 || gw.totalCaptured() != 41000 {
		t.Fatalf("capture calls = %d, total = %d, want 1 and 41000", gw.captureCall, gw.totalCaptured())
	}
}
