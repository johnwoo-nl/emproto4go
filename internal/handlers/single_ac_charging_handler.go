package handlers

import (
	"encoding/binary"
	"time"

	impl "github.com/johnwoo-nl/emproto4go/internal"
	"github.com/johnwoo-nl/emproto4go/types"
)

type SingleAcChargingHandler struct{}

func (h SingleAcChargingHandler) Handles() []impl.EmCommand {
	return []impl.EmCommand{impl.CmdSingleACChargingPublicAuto, impl.CmdSingleACChargingStatusResponse}
}

func (h SingleAcChargingHandler) Handle(evse *impl.Evse, datagram *impl.Datagram) {
	if impl.CheckPayloadLength(datagram, evse, 74) {
		return
	}

	charge := evse.MutableCharge()
	changed := false
	now := time.Now()

	if impl.CompareAndSet(&charge.Port_, datagram.Payload[0]) {
		changed = true
	}

	var chargeState types.EmCurrentState
	if len(datagram.Payload) <= 74 || !(datagram.Payload[74] == 18 || datagram.Payload[74] == 19) {
		chargeState = types.EmCurrentState(datagram.Payload[1])
	} else {
		chargeState = types.EmCurrentState(datagram.Payload[74])
	}
	if impl.CompareAndSet(&charge.ChargeState_, chargeState) {
		changed = true
	}
	if impl.CompareAndSet(&charge.ChargeId_, types.ChargeId(impl.ReadString(datagram.Payload[2:18]))) {
		changed = true
	}
	if impl.CompareAndSet(&charge.StartType_, datagram.Payload[18]) {
		changed = true
	}
	if impl.CompareAndSet(&charge.ChargeType_, datagram.Payload[19]) {
		changed = true
	}
	if impl.CompareAndSet(&charge.MaxDuration_, impl.ReadDurationMinutes(datagram.Payload, 20)) {
		changed = true
	}
	if impl.CompareAndSet(&charge.MaxEnergy_, impl.ReadEnergy16(datagram.Payload, 22)) {
		changed = true
	}
	if impl.CompareAndSet(&charge.ReservationTime_, impl.ReadTimestamp(datagram.Payload, 26)) {
		changed = true
	}
	if impl.CompareAndSet(&charge.UserId_, types.UserId(impl.ReadString(datagram.Payload[30:46]))) {
		changed = true
	}
	if impl.CompareAndSet(&charge.MaxCurrent_, types.Amps(datagram.Payload[46])) {
		changed = true
	}
	if impl.CompareAndSet(&charge.StartTime_, impl.ReadTimestamp(datagram.Payload, 47)) {
		changed = true
	}
	if impl.CompareAndSet(&charge.Duration_, impl.ReadDurationSeconds(datagram.Payload, 51)) {
		changed = true
	}
	if impl.CompareAndSet(&charge.StartEnergyCounter_, *impl.ReadEnergy32(datagram.Payload, 55)) {
		changed = true
	}
	if impl.CompareAndSet(&charge.CurrentEnergyCounter_, *impl.ReadEnergy32(datagram.Payload, 59)) {
		changed = true
	}
	if impl.CompareAndSet(&charge.ChargedEnergy_, *impl.ReadEnergy32(datagram.Payload, 63)) {
		changed = true
	}
	if impl.CompareAndSet(&charge.ChargePrice_, float32(binary.BigEndian.Uint32(datagram.Payload[67:71]))*0.01) {
		changed = true
	}
	if impl.CompareAndSet(&charge.FeeType_, datagram.Payload[71]) {
		changed = true
	}
	if impl.CompareAndSet(&charge.ChargeFee_, float32(binary.BigEndian.Uint16(datagram.Payload[72:74]))*0.01) {
		changed = true
	}

	charge.LastFetched = &now

	response := &impl.Datagram{
		Command: impl.CmdSingleACChargingAck,
		Payload: []byte{0},
	}
	go func() {
		err := evse.SendDatagram(response)
		if err != nil {
			evse.Communicator().Logger_.Warnf("[emproto4go] Failed to send SingleACChargingResponse to EVSE %s: %v.",
				evse.Serial(), err)
		}
	}()

	if changed {
		evse.QueueEvent(types.EvseChargeUpdated)
	}
}

func init() {
	impl.HandlerDelegator.Register(
		SingleAcChargingHandler{},
	)
}
