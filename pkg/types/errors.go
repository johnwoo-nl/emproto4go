package types

import (
	"fmt"
)

type EvseOfflineError struct {
	Evse EmEvse
}

func (err EvseOfflineError) Error() string {
	return fmt.Sprintf("EVSE is offline: %s", err.Evse.Label())
}

type EvseOnlineError struct {
	Evse EmEvse
}

func (err EvseOnlineError) Error() string {
	return fmt.Sprintf("EVSE is online: %s", err.Evse.Label())
}

type EvseNotLoggedInError struct {
	Evse EmEvse
}

func (err EvseNotLoggedInError) Error() string {
	return fmt.Sprintf("EVSE is not logged in: %s", err.Evse.Label())
}

type EvseNoPasswordError struct {
	Evse EmEvse
}

func (err EvseNoPasswordError) Error() string {
	return fmt.Sprintf("No password for EVSE: %s", err.Evse.Label())
}

type EvseInvalidPasswordError struct {
	Evse EmEvse
}

func (err EvseInvalidPasswordError) Error() string {
	return fmt.Sprintf("Invalid password for EVSE: %s", err.Evse.Label())
}

type EvseInvalidDatagramError struct {
	Evse            EmEvse
	ResponseCommand uint16
}

func (err EvseInvalidDatagramError) Error() string {
	return fmt.Sprintf("Invalid response for %04x for EVSE: %s", err.ResponseCommand, err.Evse.Label())
}

type EvseConfigFetchError struct {
	Evse         EmEvse
	FailedFields []string
}

func (err EvseConfigFetchError) Error() string {
	return fmt.Sprintf("Failed to get some configuration fields for EVSE %s: %v", err.Evse.Label(), err.FailedFields)
}

type EvseChargeStartError struct {
	Evse         EmEvse
	ErrorReason  ChargeStartErrorReason
	ErrorMessage string
}

func (err EvseChargeStartError) Error() string {
	return fmt.Sprintf("Failed to start charge for EVSE %s: %s (%d)", err.Evse.Label(), err.ErrorMessage, err.ErrorReason)
}

type EvseChargeStopError struct {
	Evse         EmEvse
	ErrorReason  ChargeStopErrorReason
	ErrorMessage string
}

func (err EvseChargeStopError) Error() string {
	return fmt.Sprintf("Failed to stop charge for EVSE %s: %s (%d)", err.Evse.Label(), err.ErrorMessage, err.ErrorReason)
}

type EvseUnknownError struct {
	Serial EmSerial
}

func (err EvseUnknownError) Error() string {
	return fmt.Sprintf("EVSE serial is unknown: %s", err.Serial)
}

type NotImplementedError struct {
	Feature string
}

func (err NotImplementedError) Error() string {
	return fmt.Sprintf("Feature not implemented: %s", err.Feature)
}
