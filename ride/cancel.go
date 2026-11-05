package ride

// CancelFee is what the rider would pay to cancel now; show it before they confirm (CANCEL-1, CANCEL-2).
func (s *Service) CancelFee(id RideID) Money {
	r := s.rides[id]
	if r.state != Assigned && r.state != Arrived {
		return 0
	}
	if s.clock.Now().Sub(r.assignedAt) <= s.cfg.CancelGrace {
		return 0
	}
	return s.cfg.CancelFee
}

// RiderCancel cancels the ride. A fee applies only if ack equals the fee the rider was shown.
func (s *Service) RiderCancel(id RideID, ack Money) (Money, error) {
	r := s.rides[id]
	s.expire(r)
	if !r.state.active() || r.state == InProgress {
		return 0, ErrBadState
	}
	fee := s.CancelFee(id)
	if fee > 0 && ack != fee {
		return 0, ErrFeeNotAcknowledged
	}
	s.end(r, Cancelled)
	s.takeFee(r, fee)
	return fee, nil
}

// takeFee charges a cancellation or no-show fee and credits 100% to the driver, no commission (CANCEL-2, CANCEL-3).
func (s *Service) takeFee(r *ride, fee Money) {
	if fee == 0 {
		return
	}
	s.chargeRider(r, string(r.id)+"-fee", fee)
	s.credit[r.driver] += fee
}

// DriverNoShow lets the driver cancel after the wait time at pickup (CANCEL-3).
func (s *Service) DriverNoShow(id RideID) (Money, error) {
	r := s.rides[id]
	if r.state != Arrived {
		return 0, ErrBadState
	}
	if s.clock.Now().Sub(r.arrivedAt) < s.cfg.NoShowWait {
		return 0, ErrTooEarly
	}
	s.end(r, Cancelled)
	s.takeFee(r, s.cfg.CancelFee)
	return s.cfg.CancelFee, nil
}

// DriverCancel frees the rider of any fee and re-dispatches to the next driver (DCANCEL-1).
func (s *Service) DriverCancel(id RideID) error {
	r := s.rides[id]
	if r.state != Assigned && r.state != Arrived {
		return ErrBadState
	}
	r.driver = ""
	s.nextOffer(r, s.clock.Now())
	return nil
}
