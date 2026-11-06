package tests

import (
	"testing"

	"cocoyashi/ride"
)

// TRIP-1, forbidden: the driver never ended a finished trip; it ends automatically and costs no more than the upfront fare.
func TestUnendedTripIsNotOvercharged(t *testing.T) {
	f := ride.Facts{AutoEnded: true, Cause: ride.DriverDetour, EstMinutes: 30, ActualMinutes: 120, Metered: 90000}
	if got := charge(t, f); got != 41000 {
		t.Fatalf("charged %d, want 41000", got)
	}
}
