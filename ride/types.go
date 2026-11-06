// Package ride holds the core trip loop of the ride-hailing app. Rule IDs in comments
// refer to ridehailing.rules.yaml; numbers that differ per market arrive through config.
package ride

import (
	"errors"
	"time"
)

// Money is whole units of the market currency. IDR has no minor unit.
type Money int64

// Set marks a market rate as set. A nil rate is "not set yet" (null in ridehailing.markets.yaml).
func Set(m Money) *Money { return &m }

// Rates is the FARE-2 slice of one market's config.
type Rates struct {
	Base      *Money // base_fare
	PerKm     *Money // per_km_rate
	PerMinute *Money // per_minute_rate
	Min       *Money // min_fare
}

type Quote struct {
	Fare      Money
	Available bool
	Normal    Money // fare before surge; zero means the fare had no surge
	CapX100   int64 // surge cap in force in hundredths; zero means none was applied
	Tolls     Money // tolls known at request time, held on top of the fare (PAY-3)
	Surged    bool  // the fare includes a demand increase and the rider is told so (SURGE-1)
}

// SurgeCaps are the limits on the demand multiple, in hundredths (200 = 2x). Zero means no limit.
type SurgeCaps struct {
	Platform int64 // surge_cap_multiple
	Legal    int64 // surge_legal_cap
}

type (
	RiderID string
	RideID  string
)

type State int

const (
	Requested State = iota
	Offering
	Assigned
	Arrived
	NoDriver
	Cancelled
	InProgress
	Completed
)

type DriverID string

type Clock interface{ Now() time.Time }

// Config is the slice of ridehailing.markets.yaml the trip loop needs, for one market.
type Config struct {
	ReportWindow         time.Duration // refund_report_days
	RefundProcess        time.Duration // refund_process_hours
	CommissionBPS        int64         // commission_rate in basis points (20% = 2000)
	CommissionCapBPS     int64         // commission_legal_cap in basis points, 0 = no cap
	DetourPercent        int64         // detour_threshold_percent
	DestChangesPerTrip   int           // destination_changes_per_trip
	MinFare              *Money        // min_fare, also the floor for a partial trip (TRIP-2)
	CancelGrace          time.Duration // cancel_grace_minutes
	NoShowWait           time.Duration // noshow_wait_minutes
	CancelFee            *Money        // cancel_fee_amount, nil = not set (CANCEL-4)
	IgnoredBeforeTimeout int           // ignored_offers_before_timeout
	IgnoredTimeout       time.Duration // ignored_offer_timeout_minutes
	OfferAccept          time.Duration // offer_accept_seconds
	SearchRadiusKm       float64       // driver_search_radius_km (MATCH-5)
	MaxSearch            time.Duration // max_search_seconds (MATCH-5)
}

// Candidate is a driver who could take the ride and how far away they are.
type Candidate struct {
	ID         DriverID
	DistanceKm float64
}

var (
	ErrNoEstimate         = errors.New("no fare estimate available")
	ErrActiveRide         = errors.New("rider already has an active ride")
	ErrOfferClosed        = errors.New("offer is not open to this driver")
	ErrFeeNotAcknowledged = errors.New("rider has not confirmed the cancellation fee")
	ErrOutstandingBalance = errors.New("rider has an unpaid balance")
	ErrNoPendingChange    = errors.New("no price change waiting for confirmation")
	ErrReportWindowClosed = errors.New("the reporting window for this trip has closed")
	ErrBadAmount          = errors.New("amount is not valid for this trip")
	ErrPaymentMethod      = errors.New("payment method expired or declined; choose another")
	ErrChangeLimit        = errors.New("destination change limit reached")
	ErrTooEarly           = errors.New("no-show wait time has not passed")
	ErrBadState           = errors.New("action not allowed in the ride's current state")
)

// Notice is a message to the rider.
type Notice string

const (
	NoticeDriverCancelled Notice = "driver cancelled; looking for another driver"
	NoticeTripEndedShort  Notice = "trip ended before the destination; you are charged for the part completed"
)

// Gateway is the card or wallet provider. Hold reserves an amount; Capture charges the final amount and
// releases the rest of the hold. Both must be idempotent per key.
type Gateway interface {
	Hold(key string, amount Money) error
	Capture(key string, amount Money) error
	Refund(key string, amount Money) error
}

type DetourCause int

const (
	NoDetour DetourCause = iota
	RiderDetour
	DriverDetour
)

// Facts are what the trip recorded, handed to CompleteWith.
type Facts struct {
	Tolls           Money // tolls not in the estimate
	Metered         Money // fare by actual time and distance
	EstMinutes      int64
	ActualMinutes   int64
	Cause           DetourCause
	EndedShort      bool  // the trip ended before the requested destination (TRIP-1)
	CompletedMeters int64 // distance done, when EndedShort
	TotalMeters     int64 // distance to the requested destination
	AutoEnded       bool  // the driver never ended the trip
}

type PayMethod int

const (
	Card PayMethod = iota
	Cash
)

type ReportID string
