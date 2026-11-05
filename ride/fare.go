package ride

// Rates are the per-market fare numbers (fare_base, fare_per_km, fare_per_minute, fare_min).
// Set is false while any of them is still null in ridehailing.markets.yaml.
type Rates struct {
	Set                         bool
	Base, PerKm, PerMinute, Min Money
}

// CalcFare returns the estimate for a route (FARE-2), before surge, tolls and promo.
// A market whose rates are not set offers no estimate, so REQ-1 blocks the request.
func CalcFare(r Rates, meters, seconds int64) (Money, error) {
	if !r.Set {
		return 0, ErrNoEstimate
	}
	fare := r.Base + r.PerKm*Money(meters)/1000 + r.PerMinute*Money(seconds)/60
	if fare < r.Min {
		fare = r.Min
	}
	return fare, nil
}
