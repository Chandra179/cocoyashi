package tests

import (
	"errors"
	"testing"

	"cocoyashi/ride"
)

var rates = ride.Rates{Set: true, Base: 8000, PerKm: 2500, PerMinute: 400, Min: 12000}

// FARE-2: base + distance + time, raised to the minimum fare.
func TestFareFromDistanceAndTime(t *testing.T) {
	got, err := ride.CalcFare(rates, 10000, 20*60)
	if err != nil || got != 41000 {
		t.Fatalf("fare=%d err=%v, want 41000", got, err)
	}
}

func TestMinimumFareApplies(t *testing.T) {
	got, _ := ride.CalcFare(rates, 1000, 3*60)
	if got != 12000 {
		t.Fatalf("fare=%d, want 12000", got)
	}
}

func TestUnsetMarketOffersNoEstimate(t *testing.T) {
	_, err := ride.CalcFare(ride.Rates{}, 10000, 1200)
	if !errors.Is(err, ride.ErrNoEstimate) {
		t.Fatalf("want ErrNoEstimate, got %v", err)
	}
}
