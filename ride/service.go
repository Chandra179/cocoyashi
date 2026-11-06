package ride

import (
	"fmt"
	"time"
)

type trip struct {
	rider       RiderID
	upfront     Money
	state       State
	driver      DriverID
	fee         Money // cancellation fee charged
	pm          PayMethod
	completedAt time.Time
	fare        Money  // final fare, once completed
	pending     *Quote // new price waiting for the rider to confirm (FARE-1 a)
	changes     int
	assigned    time.Time
	arrived     time.Time
	queue       []DriverID // drivers still to be offered, the first has the open offer
	offeredAt   time.Time
	searchBeg   time.Time // when dispatch started, for the maximum search time (MATCH-5)
}

// Service is the trip loop. It is in memory, with no database or HTTP layer.
// Offer timeouts are evaluated lazily whenever a ride is read or acted on, so there is no timer.
type Service struct {
	cfg      Config
	clock    Clock
	gw       Gateway
	trips    map[RideID]*trip
	active   map[RiderID]RideID // a rider's ride until it ends (REQ-2)
	streak   map[DriverID]int   // offers ignored in a row (MATCH-3)
	offUntil map[DriverID]time.Time
	reports  map[ReportID]*report
	notices  map[RiderID][]Notice
	owed     map[RiderID]Money
	comOwed  map[DriverID]Money // commission drivers owe on cash trips (PAYM-1)
	credit   map[DriverID]Money // fees credited to drivers, no commission taken (CANCEL-2)
}

func NewService(cfg Config, clock Clock, gw Gateway) *Service {
	return &Service{cfg: cfg, clock: clock, gw: gw, trips: map[RideID]*trip{}, active: map[RiderID]RideID{}, credit: map[DriverID]Money{}, owed: map[RiderID]Money{}, comOwed: map[DriverID]Money{}, notices: map[RiderID][]Notice{}, reports: map[ReportID]*report{}, streak: map[DriverID]int{}, offUntil: map[DriverID]time.Time{}}
}

func (s *Service) State(id RideID) State   { s.expire(s.trips[id]); return s.trips[id].state }
func (s *Service) Upfront(id RideID) Money { return s.trips[id].upfront }

// RequestRide creates a request that carries the estimate the rider saw (REQ-1).
func (s *Service) RequestRide(rider RiderID, q Quote) (RideID, error) {
	return s.RequestRideWith(rider, q, Card)
}

// RequestRideWith is RequestRide for a chosen payment method (PAYM-1).
func (s *Service) RequestRideWith(rider RiderID, q Quote, pm PayMethod) (RideID, error) {
	if !q.Available {
		return "", ErrNoEstimate
	}
	if s.owed[rider] > 0 {
		return "", ErrOutstandingBalance
	}
	if _, busy := s.active[rider]; busy {
		return "", ErrActiveRide
	}
	id := RideID(fmt.Sprintf("ride-%d", len(s.trips)+1))
	if pm == Card { // a cash trip places no hold (PAYM-1)
		if err := s.gw.Hold(string(id), holdAmount(q)); err != nil {
			return "", ErrPaymentMethod
		}
	}
	s.trips[id] = &trip{rider: rider, upfront: q.Fare, state: Requested, pm: pm}
	s.active[rider] = id
	return id, nil
}

// Dispatch offers the ride to the first driver; only one offer is open at a time (MATCH-1).
func (s *Service) Dispatch(id RideID, drivers []Candidate) {
	t := s.trips[id]
	for _, c := range drivers {
		if !s.IsOnline(c.ID) {
			continue
		}
		if s.cfg.SearchRadiusKm > 0 && c.DistanceKm > s.cfg.SearchRadiusKm {
			continue
		}
		t.queue = append(t.queue, c.ID)
	}
	t.state = Offering
	t.offeredAt = s.clock.Now()
	t.searchBeg = t.offeredAt
}

// OfferedTo reports the one driver holding the open offer, if any.
func (s *Service) OfferedTo(id RideID) (DriverID, bool) {
	t := s.trips[id]
	s.expire(t)
	if t.state != Offering || len(t.queue) == 0 {
		return "", false
	}
	return t.queue[0], true
}

// expire closes offers whose accept time has run out and passes them on (MATCH-2).
func (s *Service) expire(t *trip) {
	if t.state == Offering && s.cfg.MaxSearch > 0 && !s.clock.Now().Before(t.searchBeg.Add(s.cfg.MaxSearch)) {
		s.end(t, NoDriver)
		return
	}
	for t.state == Offering && len(t.queue) > 0 && !s.clock.Now().Before(t.offeredAt.Add(s.cfg.OfferAccept)) {
		t.offeredAt = t.offeredAt.Add(s.cfg.OfferAccept)
		s.ignored(t.queue[0], t.offeredAt)
		t.queue = t.queue[1:]
	}
	if t.state == Offering && len(t.queue) == 0 {
		s.end(t, NoDriver)
	}
}

// end closes a ride and frees the rider to request again (REQ-2).
func (s *Service) end(t *trip, to State) {
	t.state = to
	delete(s.active, t.rider)
}

// Accept assigns the ride to the driver holding the open offer, and to nobody else (MATCH-1, MATCH-2).
func (s *Service) Accept(id RideID, d DriverID) error {
	t := s.trips[id]
	s.expire(t)
	if t.state != Offering || len(t.queue) == 0 || t.queue[0] != d {
		return ErrOfferClosed
	}
	t.state = Assigned
	t.driver = d
	s.streak[d] = 0 // accepting breaks the run of ignored offers (MATCH-3)
	t.assigned = s.clock.Now()
	return nil
}

// Decline passes the open offer to the next driver at once.
func (s *Service) Decline(id RideID, d DriverID) {
	t := s.trips[id]
	s.expire(t)
	if t.state != Offering || len(t.queue) == 0 || t.queue[0] != d {
		return
	}
	s.ignored(d, s.clock.Now())
	t.queue = t.queue[1:]
	t.offeredAt = s.clock.Now()
}

func (s *Service) Driver(id RideID) DriverID { return s.trips[id].driver }

// Cancel ends the ride at the rider's request (CANCEL-1).
func (s *Service) Cancel(id RideID, ackFee Money) error {
	t := s.trips[id]
	s.expire(t)
	fee := s.cancelFee(t)
	if ackFee != fee {
		return ErrFeeNotAcknowledged
	}
	t.fee = fee
	s.credit[t.driver] += fee
	s.end(t, Cancelled)
	return nil
}

// CancelQuote is the fee the rider would pay if they cancelled now, shown before they confirm (CANCEL-2).
func (s *Service) CancelQuote(id RideID) Money {
	t := s.trips[id]
	s.expire(t)
	return s.cancelFee(t)
}

func (s *Service) cancelFee(t *trip) Money {
	if (t.state != Assigned && t.state != Arrived) || s.cfg.CancelFee == nil {
		return 0
	}
	if s.clock.Now().Before(t.assigned.Add(s.cfg.CancelGrace)) {
		return 0
	}
	return *s.cfg.CancelFee
}

func (s *Service) DriverCredit(d DriverID) Money { return s.credit[d] }

func (s *Service) FeeCharged(id RideID) Money { return s.trips[id].fee }

// Arrive records that the driver reached the pickup point.
func (s *Service) Arrive(id RideID, d DriverID) error {
	t := s.trips[id]
	if t.state != Assigned || t.driver != d {
		return ErrBadState
	}
	t.state = Arrived
	t.arrived = s.clock.Now()
	return nil
}

// NoShowCancel lets the driver cancel after waiting; the rider pays the same fee as CANCEL-2 (CANCEL-3).
func (s *Service) NoShowCancel(id RideID, d DriverID) error {
	t := s.trips[id]
	if t.state != Arrived || t.driver != d {
		return ErrBadState
	}
	if s.clock.Now().Before(t.arrived.Add(s.cfg.NoShowWait)) {
		return ErrTooEarly
	}
	if s.cfg.CancelFee != nil {
		t.fee = *s.cfg.CancelFee
		s.credit[d] += t.fee
	}
	s.end(t, Cancelled)
	return nil
}

// DriverCancel is a driver backing out after accepting: no fee, and the request goes to the next driver (DCANCEL-1).
func (s *Service) DriverCancel(id RideID, d DriverID) error {
	t := s.trips[id]
	if (t.state != Assigned && t.state != Arrived) || t.driver != d {
		return ErrBadState
	}
	s.notices[t.rider] = append(s.notices[t.rider], NoticeDriverCancelled)
	t.queue = t.queue[1:]
	t.driver = ""
	t.state = Offering
	t.offeredAt = s.clock.Now()
	return nil
}

func (s *Service) Notices(r RiderID) []Notice { return s.notices[r] }

// Start begins the trip once the driver has the rider.
func (s *Service) Start(id RideID, d DriverID) error {
	t := s.trips[id]
	if t.state != Arrived || t.driver != d {
		return ErrBadState
	}
	t.state = InProgress
	return nil
}

// Complete ends a trip that went as quoted.
func (s *Service) Complete(id RideID) error { return s.CompleteWith(id, Facts{}) }

// CompleteWith ends the trip and charges the final fare once (PAY-1, FARE-1).
func (s *Service) CompleteWith(id RideID, f Facts) error {
	t := s.trips[id]
	if t.state != InProgress {
		return ErrBadState
	}
	fare := finalFare(t.upfront, f, s.cfg.DetourPercent, s.cfg.MinFare)
	t.fare = fare
	t.completedAt = s.clock.Now()
	bps := s.cfg.CommissionBPS
	if s.cfg.CommissionCapBPS > 0 && bps > s.cfg.CommissionCapBPS {
		bps = s.cfg.CommissionCapBPS // EARN-1: never above the legal cap
	}
	commission := fare * Money(bps) / 10000
	switch t.pm {
	case Cash: // the driver holds the cash, so the commission is owed by the driver (PAYM-1)
		s.comOwed[t.driver] += commission
	default:
		if err := s.gw.Capture(string(id), fare); err != nil {
			s.owed[t.rider] += fare // PAY-2: the rider owes it, the driver is still paid
		}
		s.credit[t.driver] += fare - commission
	}
	if f.EndedShort {
		s.notices[t.rider] = append(s.notices[t.rider], NoticeTripEndedShort)
	}
	s.end(t, Completed)
	return nil
}

func (s *Service) Owed(r RiderID) Money { return s.owed[r] }

// PayOutstanding retries the unpaid balance; the rider can request again once it is paid.
func (s *Service) PayOutstanding(r RiderID) error {
	if err := s.gw.Capture("owed-"+string(r), s.owed[r]); err != nil {
		return err
	}
	delete(s.owed, r)
	return nil
}

// ChangeDestination quotes a new price; it applies only after ConfirmPrice (FARE-1 a).
func (s *Service) ChangeDestination(id RideID, q Quote) error {
	t := s.trips[id]
	if t.state != InProgress && t.state != Arrived && t.state != Assigned {
		return ErrBadState
	}
	if !q.Available {
		return ErrNoEstimate
	}
	if t.changes >= s.cfg.DestChangesPerTrip {
		return ErrChangeLimit
	}
	t.pending = &q
	return nil
}

// ConfirmPrice applies the quoted price the rider accepted and uses up one change.
func (s *Service) ConfirmPrice(id RideID) error {
	t := s.trips[id]
	if t.pending == nil {
		return ErrNoPendingChange
	}
	t.upfront = t.pending.Fare
	_ = s.gw.Hold(string(id), holdAmount(*t.pending)) // the hold follows the confirmed price (PAY-3)
	t.pending = nil
	t.changes++
	return nil
}

func (s *Service) CommissionOwed(d DriverID) Money { return s.comOwed[d] }

// ReportUnpaid is a driver saying the rider did not pay a cash fare: the rider owes it and is blocked (PAYM-1).
func (s *Service) ReportUnpaid(id RideID, d DriverID) error {
	t := s.trips[id]
	if t.state != Completed || t.pm != Cash || t.driver != d {
		return ErrBadState
	}
	s.owed[t.rider] += t.fare
	return nil
}

// UpholdReport reimburses the driver's fare after a report is upheld (PAYM-1).
func (s *Service) UpholdReport(id RideID) error {
	t := s.trips[id]
	if t.state != Completed || t.pm != Cash {
		return ErrBadState
	}
	s.credit[t.driver] += t.fare
	return nil
}

// RunPayout pays every driver what they have earned and resets their balance (EARN-1).
// The caller runs it on the market's payout day.
func (s *Service) RunPayout() map[DriverID]Money {
	paid := map[DriverID]Money{}
	for d, amount := range s.credit {
		paid[d] = amount
		s.credit[d] = 0
	}
	return paid
}

type report struct {
	ride   RideID
	amount Money
	closed bool
}

// ReportFare is a rider disputing a fare. Reports are accepted inside the reporting window and always answered (REFUND-1).
func (s *Service) ReportFare(id RideID, amount Money) (ReportID, error) {
	t := s.trips[id]
	if t.state != Completed {
		return "", ErrBadState
	}
	if s.clock.Now().After(t.completedAt.Add(s.cfg.ReportWindow)) {
		return "", ErrReportWindowClosed
	}
	if amount <= 0 || amount > t.fare {
		return "", ErrBadAmount
	}
	rid := ReportID(fmt.Sprintf("report-%d", len(s.reports)+1))
	s.reports[rid] = &report{ride: id, amount: amount}
	return rid, nil
}

// ResolveReport closes a report; an upheld one is refunded at once, well inside the processing time.
func (s *Service) ResolveReport(rid ReportID, upheld bool) error {
	r := s.reports[rid]
	if r == nil || r.closed {
		return ErrBadState
	}
	r.closed = true
	if upheld {
		return s.gw.Refund(string(r.ride), r.amount)
	}
	return nil
}

// ignored counts an offer the driver did not answer; enough in a row take the driver offline (MATCH-3).
func (s *Service) ignored(d DriverID, at time.Time) {
	s.streak[d]++
	if s.cfg.IgnoredBeforeTimeout > 0 && s.streak[d] >= s.cfg.IgnoredBeforeTimeout {
		s.offUntil[d] = at.Add(s.cfg.IgnoredTimeout)
		s.streak[d] = 0
	}
}

func (s *Service) IsOnline(d DriverID) bool { return !s.clock.Now().Before(s.offUntil[d]) }
