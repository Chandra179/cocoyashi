// Package ride holds the core trip loop of the ride-hailing app. Rule IDs in comments
// refer to ridehailing.rules.yaml; numbers that differ per market arrive through Config.
package ride

import (
	"errors"
	"time"
)

type (
	Money    int64 // minor units of the market currency
	RiderID  string
	DriverID string
	RideID   string
)

type PayMethod int

const (
	Card PayMethod = iota
	Cash
)

type State int

const (
	Requested State = iota
	Offering
	Assigned
	Arrived
	InProgress
	Completed
	Cancelled
	NoDriver
)

func (s State) active() bool { return s <= InProgress }

var (
	ErrNoEstimate         = errors.New("no fare estimate available")
	ErrActiveRide         = errors.New("rider already has an active ride")
	ErrOutstandingBalance = errors.New("rider has an unpaid balance")
	ErrOfferClosed        = errors.New("offer is not open to this driver")
	ErrFeeNotAcknowledged = errors.New("rider has not confirmed the cancellation fee")
	ErrTooEarly           = errors.New("no-show wait time has not passed")
	ErrChangeLimit        = errors.New("destination change limit reached")
	ErrBadState           = errors.New("action not allowed in the ride's current state")
	ErrNoPendingChange    = errors.New("no price change waiting for confirmation")
)

type Clock interface{ Now() time.Time }

// Gateway charges a rider. Charge must be idempotent per key.
type Gateway interface {
	Charge(key string, amount Money) error
}

// Config is the slice of ridehailing.markets.yaml the trip loop needs, for one market.
type Config struct {
	CancelGrace        time.Duration // cancel_grace_minutes
	NoShowWait         time.Duration // noshow_wait_minutes
	OfferAccept        time.Duration // offer_accept_seconds
	CancelFee          Money         // cancel_fee_amount
	CommissionBPS      int64         // commission_rate in basis points (20% = 2000)
	DetourPercent      int64         // detour_threshold_percent
	DestChangesPerTrip int           // destination_changes_per_trip
}

type Quote struct {
	Fare      Money
	Available bool
}

type DetourCause int

const (
	NoDetour DetourCause = iota
	RiderDetour
	DriverDetour
)

// Facts are what the trip recorded, handed to Settle.
type Facts struct {
	Tolls                  Money // tolls not in the estimate
	Metered                Money // fare by actual time and distance
	EstMinutes             int64
	ActualMinutes          int64
	Cause                  DetourCause
	RouteChangedOutsideApp bool
	EndedShort             bool
	PartialFare            Money // fare for the part completed, when EndedShort
	AutoEnded              bool  // the driver never ended the trip
}

type Settlement struct {
	Fare          Money
	Charged       bool
	DriverEarning Money
}
