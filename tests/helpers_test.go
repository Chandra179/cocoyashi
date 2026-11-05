package tests

import (
	"errors"
	"testing"
	"time"

	"cocoyashi/ride"
)

type fakeClock struct{ t time.Time }

func (c *fakeClock) Now() time.Time          { return c.t }
func (c *fakeClock) Advance(d time.Duration) { c.t = c.t.Add(d) }

type fakeGateway struct {
	fail  bool
	calls int
	paid  map[string]ride.Money
}

func (g *fakeGateway) Charge(key string, amount ride.Money) error {
	g.calls++
	if g.fail {
		return errors.New("gateway down")
	}
	if g.paid == nil {
		g.paid = map[string]ride.Money{}
	}
	g.paid[key] = amount
	return nil
}

// cfg mirrors the defaults in ridehailing.markets.yaml, with Indonesian rupiah amounts.
func cfg() ride.Config {
	return ride.Config{
		CancelGrace:        5 * time.Minute,
		NoShowWait:         5 * time.Minute,
		OfferAccept:        15 * time.Second,
		CancelFee:          3000,
		CommissionBPS:      2000,
		DetourPercent:      30,
		DestChangesPerTrip: 1,
	}
}

func newSvc() (*ride.Service, *fakeClock, *fakeGateway) {
	clk := &fakeClock{t: time.Date(2026, 11, 5, 9, 0, 0, 0, time.UTC)}
	gw := &fakeGateway{}
	return ride.NewService(cfg(), clk, gw), clk, gw
}

func quote(fare ride.Money) ride.Quote { return ride.Quote{Fare: fare, Available: true} }

func mustRequest(t *testing.T, s *ride.Service, rider ride.RiderID, fare ride.Money, pm ride.PayMethod) ride.RideID {
	t.Helper()
	id, err := s.RequestRide(rider, quote(fare), pm)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	return id
}

// assigned requests a ride and has driver d1 accept it.
func assigned(t *testing.T, s *ride.Service, rider ride.RiderID, fare ride.Money, pm ride.PayMethod) ride.RideID {
	t.Helper()
	id := mustRequest(t, s, rider, fare, pm)
	if err := s.Dispatch(id, []ride.DriverID{"d1", "d2", "d3"}); err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	if err := s.Accept(id, "d1"); err != nil {
		t.Fatalf("accept: %v", err)
	}
	return id
}

// inProgress takes an assigned ride through arrival to the start of the trip.
func inProgress(t *testing.T, s *ride.Service, rider ride.RiderID, fare ride.Money, pm ride.PayMethod) ride.RideID {
	t.Helper()
	id := assigned(t, s, rider, fare, pm)
	if err := s.Arrive(id); err != nil {
		t.Fatalf("arrive: %v", err)
	}
	if err := s.Start(id); err != nil {
		t.Fatalf("start: %v", err)
	}
	return id
}
