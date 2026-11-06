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

// cfg mirrors the defaults in ridehailing.markets.yaml.
func cfg() ride.Config {
	// SearchRadiusKm and MaxSearch are null in the markets file; these are test values.
	return ride.Config{
		CommissionBPS: 2000, IgnoredBeforeTimeout: 3, IgnoredTimeout: 5 * time.Minute, ReportWindow: 30 * 24 * time.Hour, RefundProcess: 48 * time.Hour, DetourPercent: 30, DestChangesPerTrip: 1, MinFare: ride.Set(12000), CancelGrace: 5 * time.Minute, NoShowWait: 5 * time.Minute, CancelFee: ride.Set(3000),
		OfferAccept: 15 * time.Second, SearchRadiusKm: 5, MaxSearch: 60 * time.Second}
}

func newSvc() (*ride.Service, *fakeClock) { return newSvcWith(cfg()) }

func newSvcWith(c ride.Config) (*ride.Service, *fakeClock) {
	s, clk, _ := newSvcFull(c)
	return s, clk
}

// fakeGateway records holds and captures. A capture with a key it has seen is not charged twice.
type fakeGateway struct {
	failCapture bool
	failHold    bool // the card is expired or declined
	holds       map[string]ride.Money
	captured    map[string]ride.Money
	captureCall int
	refunded    map[string]ride.Money
}

func (g *fakeGateway) Refund(key string, amount ride.Money) error {
	if g.refunded == nil {
		g.refunded = map[string]ride.Money{}
	}
	g.refunded[key] += amount
	return nil
}

func (g *fakeGateway) Hold(key string, amount ride.Money) error {
	if g.failHold {
		return errors.New("card declined")
	}
	if g.holds == nil {
		g.holds = map[string]ride.Money{}
	}
	g.holds[key] = amount
	return nil
}

func (g *fakeGateway) Capture(key string, amount ride.Money) error {
	g.captureCall++
	if g.failCapture {
		return errors.New("gateway down")
	}
	if g.captured == nil {
		g.captured = map[string]ride.Money{}
	}
	g.captured[key] = amount
	return nil
}

func (g *fakeGateway) totalCaptured() (sum ride.Money) {
	for _, a := range g.captured {
		sum += a
	}
	return
}

func newSvcFull(c ride.Config) (*ride.Service, *fakeClock, *fakeGateway) {
	clk := &fakeClock{t: time.Date(2026, 11, 6, 9, 0, 0, 0, time.UTC)}
	gw := &fakeGateway{}
	return ride.NewService(c, clk, gw), clk, gw
}

// near makes candidates all at the pickup point.
func near(ids ...ride.DriverID) []ride.Candidate {
	var c []ride.Candidate
	for _, id := range ids {
		c = append(c, ride.Candidate{ID: id})
	}
	return c
}

// assigned takes a fresh ride to the Assigned state with driver d1.
func assigned(t *testing.T, s *ride.Service) ride.RideID {
	t.Helper()
	id, err := s.RequestRide("rider-1", quote(41000))
	if err != nil {
		t.Fatal(err)
	}
	s.Dispatch(id, near("d1"))
	if err := s.Accept(id, "d1"); err != nil {
		t.Fatal(err)
	}
	return id
}
