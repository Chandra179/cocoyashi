package ride

// ProposeChange records a new price for a destination change or added stop; it is not charged until confirmed (FARE-1a).
func (s *Service) ProposeChange(id RideID, newFare Money) error {
	r := s.rides[id]
	if r.state != InProgress {
		return ErrBadState
	}
	if r.changes >= s.cfg.DestChangesPerTrip {
		return ErrChangeLimit
	}
	r.pending = &newFare
	return nil
}

func (s *Service) ConfirmChange(id RideID) error {
	r := s.rides[id]
	if r.pending == nil {
		return ErrNoPendingChange
	}
	r.upfront, r.pending = *r.pending, nil
	r.changes++
	return nil
}

func (s *Service) overDetour(f Facts) bool {
	return f.ActualMinutes*100 > f.EstMinutes*(100+s.cfg.DetourPercent)
}

func min(a, b Money) Money {
	if a < b {
		return a
	}
	return b
}

// Fare applies FARE-1 and TRIP-1 to what the trip recorded.
func (s *Service) fare(r *ride, f Facts) Money {
	fare := r.upfront
	switch {
	case f.EndedShort:
		fare = min(f.PartialFare, r.upfront)
	case f.AutoEnded:
		// a trip the driver forgot to end never costs more than the upfront fare
	case f.RouteChangedOutsideApp, f.Cause == RiderDetour && s.overDetour(f):
		fare = f.Metered
	}
	return fare + f.Tolls
}

// Settle ends the trip, charges the rider once and pays the driver (PAY-1, PAY-2, EARN-1).
// Calling it again returns the first result without charging again.
func (s *Service) Settle(id RideID, f Facts) (Settlement, error) {
	r := s.rides[id]
	if r.settled != nil {
		return *r.settled, nil
	}
	if r.state != InProgress {
		return Settlement{}, ErrBadState
	}
	fare := s.fare(r, f)
	earning := fare * Money(10000-s.cfg.CommissionBPS) / 10000
	st := Settlement{Fare: fare, DriverEarning: earning}
	if r.pay == Card {
		st.Charged = s.chargeRider(r, string(r.id), fare)
		s.credit[r.driver] += earning // paid even if the rider's payment failed
	} else {
		s.owesCom[r.driver] += fare - earning // cash: commission collected from the driver
	}
	s.end(r, Completed)
	r.settled = &st
	return st, nil
}
