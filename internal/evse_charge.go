package internal

import (
	"log"
	"time"

	"github.com/johnwoo-nl/emproto4go/types"
)

type EvseCharge struct {
	evse        *Evse
	LastFetched *time.Time

	Port_                 uint8
	ChargeState_          types.EmCurrentState
	ChargeId_             types.ChargeId
	StartType_            uint8
	ChargeType_           uint8
	MaxDuration_          *time.Duration
	MaxEnergy_            *types.KWh
	ReservationTime_      *time.Time
	UserId_               types.UserId
	MaxCurrent_           types.Amps
	StartTime_            *time.Time
	Duration_             time.Duration
	StartEnergyCounter_   types.KWh
	CurrentEnergyCounter_ types.KWh
	ChargedEnergy_        types.KWh
	ChargePrice_          float32
	FeeType_              uint8
	ChargeFee_            float32
}

func (charge EvseCharge) Port() uint8 {
	return charge.Port_
}

func (charge EvseCharge) ChargeState() types.EmCurrentState {
	return charge.ChargeState_
}

func (charge EvseCharge) ChargeId() types.ChargeId {
	return charge.ChargeId_
}

func (charge EvseCharge) StartType() uint8 {
	return charge.StartType_
}

func (charge EvseCharge) ChargeType() uint8 {
	return charge.ChargeType_
}

func (charge EvseCharge) MaxDuration() *time.Duration {
	return charge.MaxDuration_
}

func (charge EvseCharge) MaxEnergy() *types.KWh {
	return charge.MaxEnergy_
}

func (charge EvseCharge) ReservationTime() *time.Time {
	return charge.ReservationTime_
}

func (charge EvseCharge) UserId() types.UserId {
	return charge.UserId_
}

func (charge EvseCharge) MaxCurrent() types.Amps {
	return charge.MaxCurrent_
}

func (charge EvseCharge) StartTime() *time.Time {
	return charge.StartTime_
}

func (charge EvseCharge) Duration() time.Duration {
	return charge.Duration_
}

func (charge EvseCharge) StartEnergyCounter() types.KWh {
	return charge.StartEnergyCounter_
}

func (charge EvseCharge) CurrentEnergyCounter() types.KWh {
	return charge.CurrentEnergyCounter_
}

func (charge EvseCharge) ChargedEnergy() types.KWh {
	return charge.ChargedEnergy_
}

func (charge EvseCharge) ChargePrice() float32 {
	return charge.ChargePrice_
}

func (charge EvseCharge) FeeType() uint8 {
	return charge.FeeType_
}

func (charge EvseCharge) ChargeFee() float32 {
	return charge.ChargeFee_
}

func (charge EvseCharge) Fetch(maxAge time.Duration) error {
	if charge.LastFetched != nil && time.Since(*charge.LastFetched) < maxAge {
		return nil
	}
	if !charge.evse.IsLoggedIn() {
		return types.EvseNotLoggedInError{Evse: charge.evse}
	}

	datagram := Datagram{Command: CmdRequestSingleACCharging, Payload: []byte{0x00}}
	if sendErr := charge.evse.SendDatagram(&datagram); sendErr != nil {
		return sendErr
	}
	_, recvErr := charge.evse.WaitForDatagram(5*time.Second, CmdSingleACChargingStatusResponse)
	if recvErr != nil && charge.evse.communicator.Debug {
		log.Printf("[!] Failed to fetch charge data for EVSE %s: %v.", charge.evse.Serial(), recvErr)
	}
	return recvErr
}
