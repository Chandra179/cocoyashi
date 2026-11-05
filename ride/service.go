package ride

import (
	"fmt"
	"time"
)

type ride struct {
	id         RideID
	rider      RiderID
	pay        PayMethod
	state      State
	upfront    Money
	pending    *Money
	changes    int
	drivers    []DriverID
	idx        int
	offerAt    time.Time
	driver     DriverID
	assignedAt time.Time
	arrivedAt  time.Time
	settled    *Settlement
}

type Service struct {
	cfg     Config
	clock   Clock
	gw      Gateway
	rides   map[RideID]*ride
	active  map[RiderID]RideID
	owed    map[RiderID]Money
	credit  map[DriverID]Money
	owesCom map[DriverID]Money
	nextID  int
	payNo   int
}

func NewService(cfg Config, clock Clock, gw Gateway) *Service {
	return &Service{
		cfg: cfg, clock: clock, gw: gw,
		rides:   map[RideID]*ride{},
		active:  map[RiderID]RideID{},
		owed:    map[RiderID]Money{},
		credit:  map[DriverID]Money{},
		owesCom: map[DriverID]Money{},
	}
}

func (s *Service) State(id RideID) State         { return s.rides[id].state }
func (s *Service) Upfront(id RideID) Money       { return s.rides[id].upfront }
func (s *Service) Owed(r RiderID) Money          { return s.owed[r] }
func (s *Service) DriverCredit(d DriverID) Money { return s.credit[d] }

// DriverOwesCommission is the commission on cash trips still to be collected from the driver (PAYM-1).
func (s *Service) DriverOwesCommission(d DriverID) Money { return s.owesCom[d] }

// RequestRide covers REQ-1 (needs an estimate), REQ-2 (one active ride) and PAY-2 (no unpaid balance).
func (s *Service) RequestRide(rider RiderID, q Quote, pm PayMethod) (RideID, error) {
	if !q.Available {
		return "", ErrNoEstimate
	}
	if _, ok := s.active[rider]; ok {
		return "", ErrActiveRide
	}
	if s.owed[rider] > 0 {
		return "", ErrOutstandingBalance
	}
	s.nextID++
	id := RideID(fmt.Sprintf("ride-%d", s.nextID))
	s.rides[id] = &ride{id: id, rider: rider, pay: pm, state: Requested, upfront: q.Fare}
	s.active[rider] = id
	return id, nil
}

func (s *Service) end(r *ride, to State) {
	r.state = to
	delete(s.active, r.rider)
}

// chargeRider takes amount from the rider by card, or records it as owed when the card fails
// or the rider pays cash.
func (s *Service) chargeRider(r *ride, key string, amount Money) bool {
	if r.pay == Card && s.gw.Charge(key, amount) == nil {
		return true
	}
	s.owed[r.rider] += amount
	return false
}

// PayOutstanding clears a rider's unpaid balance so new requests are allowed again (PAY-2).
func (s *Service) PayOutstanding(rider RiderID) error {
	amount := s.owed[rider]
	if amount == 0 {
		return nil
	}
	s.payNo++
	if err := s.gw.Charge(fmt.Sprintf("owed-%s-%d", rider, s.payNo), amount); err != nil {
		return err
	}
	s.owed[rider] = 0
	return nil
}
