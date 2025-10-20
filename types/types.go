package types

import (
	"net"
	"time"

	"github.com/sirupsen/logrus"
)

// EmSerial represents the serial number of an EVSE. This is a string of 16 hexadecimal characters.
type EmSerial string

// EmPassword represents the password of an EVSE. This is a string of 6 digits.
type EmPassword string

// EmCommunicator represents a communicator with a network presence that can manage multiple EmEvse instances.
type EmCommunicator interface {
	// AppName returns the app name as specified in createCommunicator. This is used as default userId in ChargeStartParams.
	AppName() UserId

	// Logger returns the logger used by this communicator.
	Logger() *logrus.Logger

	// SetLogger sets a custom logger function for this communicator. Communicator's logger will delegate log messages to this function.
	SetLogger(delegate func(string, string))

	// Start starts the communicator, which will begin discovering EVSEs.
	Start() error

	// Stop stops the communicator, which will stop discovering EVSEs and close any open connections. Any EVSEs
	// on this communicator will be considered offline since the communicator itself is offline.
	Stop()

	// GetEvse returns the EmEvse instance for the given serial number.
	GetEvse(serial EmSerial) EmEvse

	// GetEvses returns a slice of all known EmEvse instances. This excludes any EVSEs that are not yet discovered.
	GetEvses() []EmEvse

	// DefineEvse defines a new EmEvse instance with the given serial number. As long as it is not discovered yet,
	// it's state fields will be empty. It can be used to set a password on already before discovery, in which case
	// that password will be used when the EVSE is discovered.
	// If an EVSE already exists in the communicator with this serial, that existing instance will be returned.
	DefineEvse(serial EmSerial) EmEvse

	// RemoveEvse removes the given EmEvse instance from the communicator. This is only possible when the EVSE is
	// offline, as otherwise it would be rediscovered immediately.
	RemoveEvse(evse EmEvse) error

	// Watch returns a watcher instance whose channel will receive EmEvent instances when events occur for any EVSE
	// managed by this communicator. The channel will be closed when the communicator is stopped. The caller must drain
	// the channel in a separate goroutine to avoid blocking the communicator.
	// Pass nil for evse to get events for all EVSEs.
	// Call Stop() on the returned watcher to stop receiving events (this will close the channel as well, unblocking
	// any goroutines waiting for data from the channel).
	Watch(evse EmEvse, eventTypes []EmEventType, channel chan<- EmEvent) EmEventWatcher
}

type EmEvse interface {
	// Serial returns the serial number of the EVSE, which is its unique identifier. This is the same as Info().Serial().
	Serial() EmSerial

	// Label returns a human-friendly label for the EVSE (configured name or model or serial, depending
	// on what info is available - this may change as info becomes available e.g. when EVSE comes online
	// or the communicator logs in).
	Label() string

	// IP returns the last known IP address of the EVSE, or nil if not yet known.
	IP() net.IP

	// Port returns the last known UDP port number that the EVSE sent a datagram from.
	Port() int

	// MetaState returns a high-level state of the EVSE, computed from various state fields.
	MetaState() EmMetaState

	// UsePassword sets the password to use for this EVSE. The password must be a string of 6 digits.
	// If the EVSE is online, this will attempt a (re-)login with this password, which may fail, in which case
	// any existing password will remain in-place. If the EVSE is offline, the password will be saved and used
	// to log in when it comes online.
	UsePassword(password EmPassword) error

	// IsOnline returns true if the EVSE is currently online.
	IsOnline() bool

	// IsLoggedIn returns true if the communicator is currently logged in to the EVSE (and the EVSE is online).
	IsLoggedIn() bool

	// Info returns static information about the EVSE, such as brand, model, hardware version, etc.
	// This info will not change over time.
	Info() EmEvseInfo

	// State returns the operational state of the EVSE, such as current power, current charging status, etc.
	State() EmEvseState

	// Charge returns information about the current, planned, or last charging session.
	Charge() EmEvseCharge

	// Config returns configuration information about the EVSE, such as configured max current, etc.
	Config() EmEvseConfig

	// StartCharge starts a charging session with the given parameters. If the EVSE is not online or not
	// logged in, or the car is not plugged in, this will fail.
	StartCharge(params ChargeStartParams) (ChargeStartResult, error)

	// StopCharge stops the current or planned charging session. If the EVSE is not online or not logged in,
	// or no charging session is active or planned, this will fail.
	StopCharge(params ChargeStopParams) (ChargeStopResult, error)

	// Watch returns a watcher whose channel will receive events for this EVSE. This is the same as calling
	// Watch() on the communicator with this EVSE as parameter.
	// Call Stop() on the returned watcher to stop receiving events (this will close the channel as well).
	Watch(eventTypes []EmEventType, ch chan<- EmEvent) EmEventWatcher
}

type EmEvseInfo interface {
	// Serial returns the serial number of the EVSE, which is its unique identifier. This is the same as EmEvse.Serial().
	Serial() EmSerial
	// EVSE brand name.
	Brand() string
	// EVSE model name.
	Model() string
	// EVSE's hardware version.
	HardwareVersion() string
	// EVSE's software version.
	SoftwareVersion() string
	// EVSE type code. Some feature's availabilities depend on this type.
	EvseType() byte
	// How many phases the EVSE supports (1 or 3).
	Phases() EmPhases
	// Whether the EVSE supports forcing 1P charging when connected on 3P.
	CanForceSinglePhase() bool
	// Maximum power the EVSE can deliver, in Watts. Computed from values of EvseType() and Byte70().
	MaxPower() Watts
	// Maximum current the EVSE can deliver (on each phase), in Amps.
	MaxCurrent() Amps
	// Supported feature flags. Meaning unknown.
	Feature() uint32
	// Supported new feature flags. Meaning unknown.
	SupportNew() uint32
	// Byte with unknown semantics at offset 70 in the SingleACStatus datagram. CanForceSinglePhase() is computed in part from this.
	Byte70() byte

	// Fetch fetches the latest info from the EVSE if it wasn't fetched less than maxAge ago.
	Fetch(maxAge time.Duration) error
}

type EmPhases int

const (
	Phases1p = EmPhases(1)
	Phases3p = EmPhases(3)
)

type EmEvseState interface {
	LineId() LineId
	CurrentPower() Watts
	EnergyCounter() KWh
	L1Voltage() Volts
	L1Current() Amps
	L2Voltage() Volts
	L2Current() Amps
	L3Voltage() Volts
	L3Current() Amps
	InnerTemp() TempCelsius
	OuterTemp() TempCelsius
	EmergencyBtnState() EmEmergencyBtnState
	CurrentState() EmCurrentState
	GunState() EmGunState
	OutputState() EmOutputState
	Errors() []EmError
	IsNewProtocol() bool
}

type EmEvseCharge interface {
	Port() uint8
	ChargeState() EmCurrentState
	ChargeId() ChargeId
	StartType() uint8
	ChargeType() uint8
	MaxDuration() *time.Duration
	MaxEnergy() *KWh
	ReservationTime() *time.Time
	UserId() UserId
	MaxCurrent() Amps
	StartTime() *time.Time
	Duration() time.Duration
	StartEnergyCounter() KWh
	CurrentEnergyCounter() KWh
	ChargedEnergy() KWh
	ChargePrice() float32
	FeeType() uint8
	ChargeFee() float32

	// Fetch fetches the latest charge info from the EVSE if it wasn't fetched less than maxAge ago.
	Fetch(maxAge time.Duration) error
}

type EmEvseConfig interface {
	// Name returns the configured name of the EVSE. May be empty if not obtained from EVSE yet.
	Name() string
	// Language returns the configured language of the EVSE. May be 0 (UnknownLanguage) if not obtained from EVSE yet.
	Language() EmLanguage
	// TemperatureUnit returns the configured temperature unit of the EVSE. May be 0 (UnknownTempUnit) if not obtained from EVSE yet.
	TemperatureUnit() EmTemperatureUnit
	// CanOfflineCharge returns whether offline charging is enabled (starting a charge from the EVSE via button, touchscreen or card). Returns false if not obtained from EVSE yet.
	CanOfflineCharge() bool
	// MaxCurrent returns the configured maximum current of the EVSE. May be 0 if not obtained from EVSE yet.
	MaxCurrent() Amps

	// Fetch fetches the latest config from the EVSE if it wasn't fetched less than maxAge ago.
	Fetch(maxAge time.Duration) error

	// SetName sets the configured name of the EVSE. The name must be a string of maximum 11 characters (truncated if longer), and the EVSE must be online and logged in.
	SetName(name string) error
	// SetLanguage sets the configured language of the EVSE. The EVSE must be online and logged in.
	SetLanguage(language EmLanguage) error
	// SetTemperatureUnit sets the configured temperature unit of the EVSE. The EVSE must be online and logged in.
	SetTemperatureUnit(unit EmTemperatureUnit) error
	// SetOfflineCharge sets whether offline charging is enabled (starting a charge from the EVSE via button, touchscreen or card). The EVSE must be online and logged in.
	SetOfflineCharge(offlineCharge bool) error
	// SetMaxCurrent sets the configured maximum current of the EVSE. The EVSE must be online and logged in.
	SetMaxCurrent(maxCurrent Amps) error
}

type ChargeId string

type UserId string // Maximum 16 ASCII characters

type EmTemperatureUnit uint8

const (
	UnknownTempUnit = EmTemperatureUnit(0)
	Celsius         = EmTemperatureUnit(1)
	Fahrenheit      = EmTemperatureUnit(2)
)

type EmLanguage int

const (
	UnknownLanguage = EmLanguage(0)
	English         = EmLanguage(1)
	Italian         = EmLanguage(2)
	German          = EmLanguage(3)
	French          = EmLanguage(4)
	Spanish         = EmLanguage(5)
	Hebrew          = EmLanguage(6)
)

type EmError uint8

const (
	RelayStickErrorL1     = EmError(0)
	RelayStickErrorL2     = EmError(1)
	RelayStickErrorL3     = EmError(2)
	Offline               = EmError(3)
	CCError               = EmError(4)
	CPError               = EmError(5)
	EmergencyStop         = EmError(6)
	OverTemperatureInner  = EmError(7)
	OverTemperatureOuter  = EmError(8)
	Unknown9              = EmError(9)
	LeakageProtection     = EmError(10)
	ShortCircuit          = EmError(11)
	OverCurrent           = EmError(12)
	Ungrounded            = EmError(13)
	OverVoltage           = EmError(14)
	LowVoltage            = EmError(15)
	InputPowerError       = EmError(25)
	MainsOverload         = EmError(26)
	DiodeShortCircuit     = EmError(27)
	RTCFailure            = EmError(28)
	FlashMemoryFailure    = EmError(29)
	EEPROMFailure         = EmError(30)
	MeteringModuleFailure = EmError(31)
)

type EmEmergencyBtnState uint8

const (
	EmergencyButton0       = EmEmergencyBtnState(0)
	EmergencyButton1       = EmEmergencyBtnState(1)
	EmergencyButton2       = EmEmergencyBtnState(2)
	EmergencyButton3       = EmEmergencyBtnState(3)
	EmergencyButton4       = EmEmergencyBtnState(4)
	EmergencyButtonUnknown = EmEmergencyBtnState(254)
)

type EmCurrentState uint8

const (
	CurrentStateUnknown0  = EmCurrentState(0)
	EvseFault             = EmCurrentState(1)
	ChargingFault2        = EmCurrentState(2)
	ChargingFault3        = EmCurrentState(3)
	CurrentStateUnknown4  = EmCurrentState(4)
	CurrentStateUnknown5  = EmCurrentState(5)
	CurrentStateUnknown6  = EmCurrentState(6)
	CurrentStateUnknown7  = EmCurrentState(7)
	CurrentStateUnknown8  = EmCurrentState(8)
	CurrentStateUnknown9  = EmCurrentState(9)
	WaitingForSwipe       = EmCurrentState(10)
	WaitingForButton      = EmCurrentState(11)
	NotConnected          = EmCurrentState(12)
	ReadyToCharge         = EmCurrentState(13)
	Charging              = EmCurrentState(14)
	Completed             = EmCurrentState(15)
	CurrentStateUnknown16 = EmCurrentState(16)
	CompletedFullCharge   = EmCurrentState(17)
	CurrentStateUnknown18 = EmCurrentState(18) // Used from longer SingleACStatus datagram (newProtocol is true), but not handled in OEM app HomeActivity so unknown what it means.
	CurrentStateUnknown19 = EmCurrentState(19) // Used from longer SingleACStatus datagram (newProtocol is true), but not handled in OEM app HomeActivity so unknown what it means.
	ChargingReservation   = EmCurrentState(20)
	CurrentStateUnknown21 = EmCurrentState(21)
	CurrentStateUnknown22 = EmCurrentState(22)
	UnknownState          = EmCurrentState(254)
)

type EmGunState uint8

const (
	GunUnknown0          = EmGunState(0)
	GunNotConnected      = EmGunState(1)
	GunConnectedUnlocked = EmGunState(2)
	GunUnknown3          = EmGunState(3)
	GunConnectedLocked   = EmGunState(4)
	GunUnknown5          = EmGunState(5)
	GunUnknown           = EmGunState(254)
)

type EmOutputState uint8

const (
	UnknownOutputState0 = EmOutputState(0)
	OutputStateCharging = EmOutputState(1)
	OutputStateIdle     = EmOutputState(2)
	UnknownOutputState3 = EmOutputState(3)
	UnknownOutputState  = EmOutputState(254)
)

type EmMetaState string

const (
	MetaStateOffline     = EmMetaState("Offline")     // EVSE is offline (not active on the network in the last few seconds).
	MetaStateNotLoggedIn = EmMetaState("NotLoggedIn") // EVSE is online but the communicator is not logged in.
	MetaStateIdle        = EmMetaState("Idle")        // EVSE is online, communicator is logged in, there are no errors, and no car is plugged in.
	MetaStatePluggedIn   = EmMetaState("PluggedIn")   // EVSE is online, communicator is logged in, there are no errors, and a car is plugged in (but not charging).
	MetaStateCharging    = EmMetaState("Charging")    // EVSE is online, communicator is logged in, there are no errors, and it is currently charging.
	MetaStateError       = EmMetaState("Error")       // EVSE is online, communicator is logged in, but there are one or more errors present.
)

type EmEventWatcher interface {
	Stop()
	IsStopped() bool
	Evse() EmEvse
	EventTypes() []EmEventType
}

type EmEvent struct {
	Type      EmEventType
	Evse      EmEvse
	Timestamp time.Time
}

type EmEventType string

const (
	EvseAdded     = EmEventType("EVSE_ADDED")
	EvseRemoved   = EmEventType("EVSE_REMOVED")
	EvseOnline    = EmEventType("EVSE_ONLINE")
	EvseOffline   = EmEventType("EVSE_OFFLINE")
	EvseLoggedIn  = EmEventType("EVSE_LOGGED_IN")
	EvseLoggedOut = EmEventType("EVSE_LOGGED_OUT")

	EvseInfoUpdated   = EmEventType("EVSE_INFO_UPDATED")
	EvseStateUpdated  = EmEventType("EVSE_STATE_UPDATED")
	EvseChargeUpdated = EmEventType("EVSE_CHARGE_UPDATED")
	EvseConfigUpdated = EmEventType("EVSE_CONFIG_UPDATED")

	EvseChargeStarted = EmEventType("EVSE_CHARGE_STARTED")
	EvseChargeStopped = EmEventType("EVSE_CHARGE_STOPPED")
)

// EvseChanged returns those EmEventTypes that represent any change in the EVSE. That is, the added/removed
// plus info/state/config updates. It excludes the others, because they all also result in one of those
// updates:
// - Online/Offline events will also result in an InfoUpdated event due to IsOnline changing;
// - LoggedIn/LoggedOut events will also result in an InfoUpdated event due to IsLoggedIn changing;
// - ChargeStarted/ChargeStopped events will also result in a StateUpdated event due to CurrentState changing.
// If you don't need to hook into those other specific events, but you just want to keep your EVSE state representation
// up-to-date (or show it in your app somewhere), you can use this set of events when calling Watch().
func EvseChanged() []EmEventType {
	return []EmEventType{EvseAdded, EvseRemoved, EvseInfoUpdated, EvseStateUpdated, EvseChargeUpdated, EvseConfigUpdated}
}

type LineId uint8
type Volts float32
type Amps float32
type Watts uint32
type KWh float64
type TempCelsius float32

// ChargeStartParams contains parameters for starting a charge session, passed to `Evse.StartCharge()`.
type ChargeStartParams struct {
	// MaxCurrent sets the maximum current to use for this charge. The value is clamped between 6A and the EVSE's `Config().MaxCurrent()` - no error is returned for an out-of-bound value. Default if not set is the EVSE's Config().MaxCurrent().
	MaxCurrent Amps
	// If Info.CanForceSinglePhase() is true, ForceSinglePhase indicates whether to use single-phase charging for this session.
	ForceSinglePhase bool
	// ChargeId is an identifier for this charge session. It must be unique for each session per day, since it is prefixed with `yyyyMMdd`. Maximum length is 8 ASCII characters (truncated if longer).
	ChargeId string
	// UserId is an identifier for the user starting this charge. Maximum length is 16 ASCII characters (truncated if longer). If not set or empty, uses `communicator.AppName()` as specified in `createCommunicator()`.
	UserId UserId
	// StartAt delay-starts a charge. The car needs to be plugged in already. If nil or in the past, charging starts immediately.
	StartAt time.Time
	// MaxDuration limits the duration of the charge session. If nil, no duration limit is set.
	MaxDuration time.Duration
	// MaxEnergy limits the energy to deliver in this charge session. If not set (or zero), no energy limit is set.
	MaxEnergy KWh
}

type ChargeStartResult struct {
	LineId       uint8
	Current      Amps
	ErrorReason  ChargeStartErrorReason
	ErrorMessage string
}

type ChargeStartErrorReason uint8

const (
	ChargeStartOK                                = ChargeStartErrorReason(0)
	ChargeStartErrorPlugNotProperlyInserted      = ChargeStartErrorReason(1)
	ChargeStartErrorSystemError                  = ChargeStartErrorReason(2)
	ChargeStartErrorAlreadyCharging              = ChargeStartErrorReason(3)
	ChargeStartErrorSystemMaintenance            = ChargeStartErrorReason(4)
	ChargeStartErrorIncorrectSetFee              = ChargeStartErrorReason(5)
	ChargeStartErrorIncorrectSetPowerConsumption = ChargeStartErrorReason(6)
	ChargeStartErrorIncorrectSetTime             = ChargeStartErrorReason(7)
	ChargeStartErrorAlreadyReserved              = ChargeStartErrorReason(20)

	ChargeStartErrorEvseOffline     = ChargeStartErrorReason(160) // Library-defined (not a protocol) error code.
	ChargeStartErrorEvseNotLoggedIn = ChargeStartErrorReason(161) // Library-defined (not a protocol) error code.
	ChargeStartErrorSendFailed      = ChargeStartErrorReason(162) // Library-defined (not a protocol) error code.
	ChargeStartErrorNoConfirmation  = ChargeStartErrorReason(163) // Library-defined (not a protocol) error code.
	ChargeStartErrorUnknown         = ChargeStartErrorReason(255) // Library-defined (not a protocol) error code.
)

// ChargeStopParams contains parameters for stopping a charge session or cancelling a planned one, passed to `Evse.ChargeStop()`.
type ChargeStopParams struct {
	LineId uint8
	// UserId is an identifier for the user stopping this charge. Maximum length is 16 ASCII characters (truncated if longer). If not set or empty, uses `communicator.AppName()` as specified in `createCommunicator()`.
	UserId UserId
}

type ChargeStopResult struct {
	LineId       uint8
	ErrorReason  ChargeStopErrorReason
	ErrorMessage string
}

type ChargeStopErrorReason uint8

const (
	ChargeStopOK = ChargeStopErrorReason(0)

	// Note: could not find possible values for ChargeStop errors, except 0 means no error.
	// There should probably be an error code indicating that no charge is active or planned, and perhaps more.
	// If you see other values in practice, please open an issue on GitHub and mention the value and semantics.

	ChargeStopErrorEvseOffline     = ChargeStopErrorReason(160) // Library-defined (not a protocol) error code.
	ChargeStopErrorEvseNotLoggedIn = ChargeStopErrorReason(161) // Library-defined (not a protocol) error code.
	ChargeStopErrorSendFailed      = ChargeStopErrorReason(162) // Library-defined (not a protocol) error code.
	ChargeStopErrorNoConfirmation  = ChargeStopErrorReason(163) // Library-defined (not a protocol) error code.
	ChargeStopErrorUnknown         = ChargeStopErrorReason(255) // Library-defined (not a protocol) error code.
)
