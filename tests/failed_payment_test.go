package tests

import (
	"errors"
	"testing"

	"cocoyashi/ride"
)

// failedTrip completes a trip whose card payment fails.
func failedTrip(t *testing.T) (*ride.Service, *fakeGateway, ride.RideID) {
	t.Helper()
	s, _, gw := newSvcFull(cfg())
	id := inProgress(t, s)
	gw.failCapture = true
	if err := s.Complete(id); err != nil {
		t.Fatalf("complete: %v", err)
	}
	return s, gw, id
}

// PAY-2, required: after a failed payment the rider owes the fare and cannot request a new ride.
func TestFailedPaymentBlocksNewRequests(t *testing.T) {
	s, _, _ := failedTrip(t)

	if got := s.Owed("rider-1"); got != 41000 {
		t.Fatalf("owed = %d, want 41000", got)
	}
	_, err := s.RequestRide("rider-1", quote(20000))
	if !errors.Is(err, ride.ErrOutstandingBalance) {
		t.Fatalf("err = %v, want ErrOutstandingBalance", err)
	}
}

// PAY-2, forbidden: the driver is still paid the fare minus commission when the rider's payment failed.
func TestDriverStillPaidWhenRiderPaymentFails(t *testing.T) {
	s, _, _ := failedTrip(t)

	if got := s.DriverCredit("d1"); got != 32800 {
		t.Fatalf("driver credit = %d, want 32800 (41000 less 20%%)", got)
	}
}

// PAY-2: paying the balance lets the rider request again.
func TestPayingTheBalanceUnblocksTheRider(t *testing.T) {
	s, gw, _ := failedTrip(t)
	gw.failCapture = false

	if err := s.PayOutstanding("rider-1"); err != nil {
		t.Fatalf("pay: %v", err)
	}

	if _, err := s.RequestRide("rider-1", quote(20000)); err != nil {
		t.Fatalf("request after paying: %v", err)
	}
}
