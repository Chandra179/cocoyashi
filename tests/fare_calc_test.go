package tests

import (
	"testing"

	"cocoyashi/ride"
)

// rates from the FARE-2 worked example: base 8000, 2500 per km, 400 per minute, minimum 12000.
func exampleRates() ride.Rates {
	return ride.Rates{Base: ride.Set(8000), PerKm: ride.Set(2500), PerMinute: ride.Set(400), Min: ride.Set(12000)}
}

// FARE-2, required: fare from distance and time.
func TestEstimateAddsBasePlusDistancePlusTime(t *testing.T) {
	q := ride.Estimate(exampleRates(), 10_000, 20)

	if !q.Available {
		t.Fatal("estimate should be available when all rates are set")
	}
	if q.Fare != 41000 {
		t.Fatalf("fare = %d, want 41000 (8000 + 25000 + 8000)", q.Fare)
	}
}

// FARE-2, required: the minimum fare applies.
func TestEstimateRaisedToMinimumFare(t *testing.T) {
	q := ride.Estimate(exampleRates(), 1_000, 3)

	if q.Fare != 12000 {
		t.Fatalf("fare = %d, want 12000, not 11700", q.Fare)
	}
}

// FARE-2, forbidden: a market with any rate unset offers no estimate.
func TestEstimateUnavailableWhenAnyRateUnset(t *testing.T) {
	for name, r := range map[string]ride.Rates{
		"base":       {PerKm: ride.Set(2500), PerMinute: ride.Set(400), Min: ride.Set(12000)},
		"per km":     {Base: ride.Set(8000), PerMinute: ride.Set(400), Min: ride.Set(12000)},
		"per minute": {Base: ride.Set(8000), PerKm: ride.Set(2500), Min: ride.Set(12000)},
		"minimum":    {Base: ride.Set(8000), PerKm: ride.Set(2500), PerMinute: ride.Set(400)},
	} {
		if q := ride.Estimate(r, 10_000, 20); q.Available {
			t.Errorf("%s unset: estimate offered (fare %d)", name, q.Fare)
		}
	}
}
