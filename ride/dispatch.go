package ride

import "time"

// Dispatch starts sequential offers to drivers in order (MATCH-1).
func (s *Service) Dispatch(id RideID, drivers []DriverID) error {
	r := s.rides[id]
	if r.state != Requested {
		return ErrBadState
	}
	r.drivers, r.idx, r.offerAt = drivers, -1, s.clock.Now()
	r.state = Offering
	s.nextOffer(r, s.clock.Now())
	return nil
}

// nextOffer moves to the next driver, opening their offer at `at`, or ends the request if none is left.
func (s *Service) nextOffer(r *ride, at time.Time) {
	r.idx++
	if r.idx >= len(r.drivers) {
		s.end(r, NoDriver)
		return
	}
	r.state, r.offerAt = Offering, at
}

// expire passes the offer along for each full accept window that has run out (MATCH-2).
func (s *Service) expire(r *ride) {
	for r.state == Offering && !s.clock.Now().Before(r.offerAt.Add(s.cfg.OfferAccept)) {
		s.nextOffer(r, r.offerAt.Add(s.cfg.OfferAccept))
	}
}

func (s *Service) CurrentOffer(id RideID) (DriverID, bool) {
	r := s.rides[id]
	s.expire(r)
	if r.state != Offering {
		return "", false
	}
	return r.drivers[r.idx], true
}

func (s *Service) Accept(id RideID, d DriverID) error {
	r := s.rides[id]
	s.expire(r)
	if r.state != Offering || r.drivers[r.idx] != d {
		return ErrOfferClosed
	}
	r.state, r.driver, r.assignedAt = Assigned, d, s.clock.Now()
	return nil
}

func (s *Service) Decline(id RideID, d DriverID) {
	r := s.rides[id]
	s.expire(r)
	if r.state == Offering && r.drivers[r.idx] == d {
		s.nextOffer(r, s.clock.Now())
	}
}

func (s *Service) Arrive(id RideID) error {
	r := s.rides[id]
	if r.state != Assigned {
		return ErrBadState
	}
	r.state, r.arrivedAt = Arrived, s.clock.Now()
	return nil
}

func (s *Service) Start(id RideID) error {
	r := s.rides[id]
	if r.state != Arrived {
		return ErrBadState
	}
	r.state = InProgress
	return nil
}
