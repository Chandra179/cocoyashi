package tests

import (
	"errors"
	"testing"

	"cocoyashi/ride"
)

func settleFare(t *testing.T, f ride.Facts, upfront ride.Money) ride.Money {
	t.Helper()
	s, _, _ := newSvc()
	id := inProgress(t, s, "r1", upfront, ride.Card)
	st, err := s.Settle(id, f)
	if err != nil {
		t.Fatal(err)
	}
	return st.Fare
}

// FARE-1: the rider pays the upfront fare, with only the listed exceptions.
func TestChargedTheEstimate(t *testing.T) {
	if got := settleFare(t, ride.Facts{Metered: 40000}, 25000); got != 25000 {
		t.Fatalf("fare = %d, want 25000 even though metered is higher", got)
	}
}

func TestTollsNotInEstimateAreAdded(t *testing.T) {
	if got := settleFare(t, ride.Facts{Tolls: 8000}, 25000); got != 33000 {
		t.Fatalf("fare = %d, want 33000", got)
	}
}

func TestConfirmedDestinationChange(t *testing.T) {
	s, _, _ := newSvc()
	id := inProgress(t, s, "r1", 25000, ride.Card)
	if err := s.ProposeChange(id, 40000); err != nil {
		t.Fatal(err)
	}
	if err := s.ConfirmChange(id); err != nil {
		t.Fatal(err)
	}
	st, _ := s.Settle(id, ride.Facts{})
	if st.Fare != 40000 {
		t.Fatalf("fare = %d, want 40000", st.Fare)
	}
}

func TestUnconfirmedChangeNotCharged(t *testing.T) {
	s, _, _ := newSvc()
	id := inProgress(t, s, "r1", 25000, ride.Card)
	s.ProposeChange(id, 40000)
	st, _ := s.Settle(id, ride.Facts{})
	if st.Fare != 25000 {
		t.Fatalf("fare = %d, want 25000: the new price was never confirmed", st.Fare)
	}
}

func TestDestinationChangeLimit(t *testing.T) {
	s, _, _ := newSvc()
	id := inProgress(t, s, "r1", 25000, ride.Card)
	s.ProposeChange(id, 40000)
	s.ConfirmChange(id)
	if err := s.ProposeChange(id, 50000); !errors.Is(err, ride.ErrChangeLimit) {
		t.Fatalf("want ErrChangeLimit, got %v", err)
	}
}

func TestDriverDetourDoesNotRaiseFare(t *testing.T) {
	f := ride.Facts{Metered: 60000, EstMinutes: 20, ActualMinutes: 40, Cause: ride.DriverDetour}
	if got := settleFare(t, f, 25000); got != 25000 {
		t.Fatalf("fare = %d, want 25000", got)
	}
}

func TestRiderDetourBeyondThresholdIsMetered(t *testing.T) {
	f := ride.Facts{Metered: 31000, EstMinutes: 20, ActualMinutes: 27, Cause: ride.RiderDetour} // 35% over
	if got := settleFare(t, f, 25000); got != 31000 {
		t.Fatalf("fare = %d, want metered 31000", got)
	}
}

func TestRiderDetourWithinThresholdKeepsUpfront(t *testing.T) {
	f := ride.Facts{Metered: 29000, EstMinutes: 20, ActualMinutes: 25, Cause: ride.RiderDetour} // 25% over
	if got := settleFare(t, f, 25000); got != 25000 {
		t.Fatalf("fare = %d, want 25000", got)
	}
}

func TestRouteChangedOutsideAppIsMetered(t *testing.T) {
	f := ride.Facts{Metered: 31000, RouteChangedOutsideApp: true}
	if got := settleFare(t, f, 25000); got != 31000 {
		t.Fatalf("fare = %d, want metered 31000", got)
	}
}
