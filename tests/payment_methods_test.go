package tests

import (
	"errors"
	"testing"

	"cocoyashi/ride"
)

// PAYM-1, forbidden: a cash request places no card hold.
func TestCashRequestPlacesNoHold(t *testing.T) {
	s, _, gw := newSvcFull(cfg())

	if _, err := s.RequestRideWith("rider-1", quote(41000), ride.Cash); err != nil {
		t.Fatalf("request: %v", err)
	}

	if len(gw.holds) != 0 {
		t.Fatalf("holds = %v, want none", gw.holds)
	}
}

// PAYM-1, forbidden: a cash request is not refused just because no hold could be placed.
func TestCashRequestNotBlockedByMissingHold(t *testing.T) {
	s, _, gw := newSvcFull(cfg())
	gw.failHold = true

	if _, err := s.RequestRideWith("rider-1", quote(41000), ride.Cash); err != nil {
		t.Fatalf("cash request refused: %v", err)
	}
}

// PAYM-1, required: a declined or expired card blocks the request and asks for another method.
func TestDeclinedCardBlocksTheRequest(t *testing.T) {
	s, _, gw := newSvcFull(cfg())
	gw.failHold = true

	id, err := s.RequestRide("rider-1", quote(41000))

	if !errors.Is(err, ride.ErrPaymentMethod) {
		t.Fatalf("err = %v, want ErrPaymentMethod", err)
	}
	if id != "" {
		t.Fatalf("ride %q created with a declined card", id)
	}
	if _, err := s.RequestRide("rider-1", quote(41000)); errors.Is(err, ride.ErrActiveRide) {
		t.Fatal("the failed request left the rider with an active ride")
	}
}

func cashTrip(t *testing.T, s *ride.Service) ride.RideID {
	t.Helper()
	id, err := s.RequestRideWith("rider-1", quote(41000), ride.Cash)
	if err != nil {
		t.Fatal(err)
	}
	s.Dispatch(id, near("d1"))
	_ = s.Accept(id, "d1")
	_ = s.Arrive(id, "d1")
	_ = s.Start(id, "d1")
	return id
}

// PAYM-1, required: on a cash trip nothing is charged to a card, and the commission is recorded as owed by the driver.
func TestCashTripRecordsCommissionOwedByDriver(t *testing.T) {
	s, _, gw := newSvcFull(cfg())
	id := cashTrip(t, s)

	if err := s.Complete(id); err != nil {
		t.Fatalf("complete: %v", err)
	}

	if gw.captureCall != 0 {
		t.Fatalf("gateway captured %d times on a cash trip", gw.captureCall)
	}
	if got := s.CommissionOwed("d1"); got != 8200 {
		t.Fatalf("commission owed = %d, want 8200 (20%% of 41000)", got)
	}
	if got := s.DriverCredit("d1"); got != 0 {
		t.Fatalf("driver credit = %d: the driver already holds the cash, want 0", got)
	}
}

// PAYM-1, required: when the rider did not pay cash and the driver reports it, the rider is blocked until paying in the app.
func TestUnpaidCashFareBlocksTheRider(t *testing.T) {
	s, _, _ := newSvcFull(cfg())
	id := cashTrip(t, s)
	_ = s.Complete(id)

	if err := s.ReportUnpaid(id, "d1"); err != nil {
		t.Fatalf("report: %v", err)
	}

	got, err := s.RequestRide("rider-1", quote(20000))
	if !errors.Is(err, ride.ErrOutstandingBalance) || got != "" {
		t.Fatalf("request = %q, %v; want blocked with ErrOutstandingBalance", got, err)
	}
}

// PAYM-1, forbidden: the debt is not added to the next fare; once paid, the next ride costs its own fare.
func TestPaidCashDebtIsNotAddedToNextFare(t *testing.T) {
	s, _, gw := newSvcFull(cfg())
	id := cashTrip(t, s)
	_ = s.Complete(id)
	_ = s.ReportUnpaid(id, "d1")
	if err := s.PayOutstanding("rider-1"); err != nil {
		t.Fatalf("pay: %v", err)
	}

	next, err := s.RequestRide("rider-1", quote(20000))
	if err != nil {
		t.Fatalf("request: %v", err)
	}

	if got := gw.holds[string(next)]; got != 20000 {
		t.Fatalf("hold = %d, want 20000 with no debt added", got)
	}
}

// PAYM-1, required: when the report is upheld the driver is reimbursed the fare.
func TestUpheldReportReimbursesTheDriver(t *testing.T) {
	s, _, _ := newSvcFull(cfg())
	id := cashTrip(t, s)
	_ = s.Complete(id)
	_ = s.ReportUnpaid(id, "d1")

	if err := s.UpholdReport(id); err != nil {
		t.Fatalf("uphold: %v", err)
	}

	if got := s.DriverCredit("d1"); got != 41000 {
		t.Fatalf("driver credit = %d, want 41000", got)
	}
}
