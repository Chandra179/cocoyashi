package tests

import (
	"errors"
	"testing"
	"time"

	"cocoyashi/ride"
)

// overcharged completes a trip charged 52000 after a rider detour that the rider disputes.
func overcharged(t *testing.T) (*ride.Service, *fakeClock, *fakeGateway, ride.RideID) {
	t.Helper()
	s, clk, gw := newSvcFull(cfg())
	id := inProgress(t, s)
	_ = s.CompleteWith(id, ride.Facts{Cause: ride.RiderDetour, EstMinutes: 30, ActualMinutes: 45, Metered: 52000})
	return s, clk, gw, id
}

// REFUND-1, required: an upheld report inside the window refunds the difference.
func TestUpheldReportIsRefunded(t *testing.T) {
	s, clk, gw, id := overcharged(t)
	clk.Advance(2 * 24 * time.Hour)

	rep, err := s.ReportFare(id, 11000)
	if err != nil {
		t.Fatalf("report: %v", err)
	}
	if err := s.ResolveReport(rep, true); err != nil {
		t.Fatalf("resolve: %v", err)
	}

	if got := gw.refunded[string(id)]; got != 11000 {
		t.Fatalf("refunded = %d, want 11000", got)
	}
}

// REFUND-1, forbidden: a report after the window is answered, not dropped.
func TestReportAfterWindowGetsAnAnswer(t *testing.T) {
	s, clk, gw, id := overcharged(t)
	clk.Advance(31 * 24 * time.Hour)

	_, err := s.ReportFare(id, 11000)

	if !errors.Is(err, ride.ErrReportWindowClosed) {
		t.Fatalf("err = %v, want ErrReportWindowClosed", err)
	}
	if len(gw.refunded) != 0 {
		t.Fatalf("refunded = %v for a late report", gw.refunded)
	}
}

// REFUND-1: a report that is not upheld refunds nothing.
func TestRejectedReportRefundsNothing(t *testing.T) {
	s, _, gw, id := overcharged(t)
	rep, _ := s.ReportFare(id, 11000)

	if err := s.ResolveReport(rep, false); err != nil {
		t.Fatalf("resolve: %v", err)
	}

	if len(gw.refunded) != 0 {
		t.Fatalf("refunded = %v, want nothing", gw.refunded)
	}
}

// REFUND-1: the last day of the window still counts.
func TestReportOnTheLastDayIsAccepted(t *testing.T) {
	s, clk, _, id := overcharged(t)
	clk.Advance(29*24*time.Hour + 23*time.Hour)

	if _, err := s.ReportFare(id, 11000); err != nil {
		t.Fatalf("report on day 30: %v", err)
	}
}

// REFUND-1: a refund can never be more than the rider was charged, and a report is refunded once.
func TestRefundCannotExceedWhatWasCharged(t *testing.T) {
	s, _, gw, id := overcharged(t)

	if _, err := s.ReportFare(id, 52001); !errors.Is(err, ride.ErrBadAmount) {
		t.Fatalf("err = %v, want ErrBadAmount", err)
	}
	rep, _ := s.ReportFare(id, 11000)
	_ = s.ResolveReport(rep, true)
	_ = s.ResolveReport(rep, true)
	if got := gw.refunded[string(id)]; got != 11000 {
		t.Fatalf("refunded = %d after resolving twice, want 11000", got)
	}
}
