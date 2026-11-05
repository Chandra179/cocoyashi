package tests

import (
	"testing"

	"cocoyashi/ride"
)

// TRIP-1: partial trips charge the part completed; unended trips never cost more than the upfront fare.
func TestPartialTripChargedPartially(t *testing.T) {
	f := ride.Facts{EndedShort: true, PartialFare: 9000}
	if got := settleFare(t, f, 25000); got != 9000 {
		t.Fatalf("fare = %d, want 9000", got)
	}
}

func TestPartialFareNeverAboveUpfront(t *testing.T) {
	f := ride.Facts{EndedShort: true, PartialFare: 30000}
	if got := settleFare(t, f, 25000); got != 25000 {
		t.Fatalf("fare = %d, want capped at 25000", got)
	}
}

func TestAutoEndedTripNotOvercharged(t *testing.T) {
	f := ride.Facts{AutoEnded: true, Metered: 70000, EstMinutes: 20, ActualMinutes: 90}
	if got := settleFare(t, f, 25000); got != 25000 {
		t.Fatalf("fare = %d, want 25000", got)
	}
}
