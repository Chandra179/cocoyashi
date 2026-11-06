package ride

// Estimate is FARE-2: base + per-km * distance + per-minute * time, before surge, tolls or promo.
func Estimate(r Rates, distanceMeters, minutes int64) Quote {
	if r.Base == nil || r.PerKm == nil || r.PerMinute == nil || r.Min == nil {
		return Quote{}
	}
	fare := *r.Base + *r.PerKm*Money(distanceMeters)/1000 + *r.PerMinute*Money(minutes)
	if fare < *r.Min {
		fare = *r.Min
	}
	return Quote{Fare: fare, Available: true}
}

// finalFare is FARE-1: the upfront fare, plus tolls, unless a rider-caused detour went beyond the threshold.
func finalFare(upfront Money, f Facts, detourPercent int64, min *Money) Money {
	if f.EndedShort && f.TotalMeters > 0 {
		return partialFare(upfront, f.CompletedMeters, f.TotalMeters, min)
	}
	if f.Cause == RiderDetour && f.EstMinutes > 0 && f.ActualMinutes*100 > f.EstMinutes*(100+detourPercent) {
		return f.Metered + f.Tolls
	}
	return upfront + f.Tolls
}

// partialFare is TRIP-2: the upfront fare pro rata by distance, at least the minimum fare, never above the upfront fare.
func partialFare(upfront Money, doneM, totalM int64, min *Money) Money {
	fare := upfront * Money(doneM) / Money(totalM)
	if min != nil && fare < *min {
		fare = *min
	}
	if fare > upfront {
		fare = upfront
	}
	return fare
}

// EstimateSurge is Estimate with demand applied (SURGE-1). demandX100 is the demand multiple in hundredths.
func EstimateSurge(r Rates, distanceMeters, minutes, demandX100 int64, caps SurgeCaps) Quote {
	q := Estimate(r, distanceMeters, minutes)
	if !q.Available {
		return q
	}
	q.Normal, q.CapX100 = q.Fare, caps.inForce()
	if demandX100 <= 100 {
		return q
	}
	if limit := caps.inForce(); limit > 0 && demandX100 > limit {
		demandX100 = limit
	}
	if demandX100 <= 100 {
		return q
	}
	q.Fare = q.Fare * Money(demandX100) / 100
	q.Surged = true
	return q
}

// inForce is the lower of the platform cap and any legal cap; zero means neither is set.
func (c SurgeCaps) inForce() int64 {
	switch {
	case c.Platform == 0:
		return c.Legal
	case c.Legal == 0:
		return c.Platform
	}
	return min(c.Platform, c.Legal)
}

// holdAmount is PAY-3: the normal fare at the surge cap in force, plus known tolls.
func holdAmount(q Quote) Money {
	normal, capX100 := q.Normal, q.CapX100
	if normal == 0 {
		normal = q.Fare
	}
	if capX100 == 0 {
		capX100 = 100
	}
	if hold := normal * Money(capX100) / 100; hold > q.Fare {
		return hold + q.Tolls
	}
	return q.Fare + q.Tolls
}
