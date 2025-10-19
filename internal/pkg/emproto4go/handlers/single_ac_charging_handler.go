package handlers

import (
	"encoding/binary"
	"log"
	"time"

	"github.com/johnwoo-nl/emproto4go/internal/pkg/emproto4go"
	"github.com/johnwoo-nl/emproto4go/internal/pkg/itypes"
	"github.com/johnwoo-nl/emproto4go/pkg/types"
)

type SingleAcChargingHandler struct{}

func (h SingleAcChargingHandler) Handles() []itypes.EmCommand {
	return []itypes.EmCommand{itypes.CmdSingleACChargingPublicAuto, itypes.CmdSingleACChargingStatusResponse}
}

func (h SingleAcChargingHandler) Handle(evse *emproto4go.Evse, datagram *emproto4go.Datagram) {
	if emproto4go.CheckPayloadLength(datagram, evse, 74) {
		return
	}

	charge := evse.MutableCharge()
	changed := false
	now := time.Now()

	if emproto4go.CompareAndSet(&charge.Port_, datagram.Payload[0]) {
		changed = true
	}

	var chargeState types.EmCurrentState
	if len(datagram.Payload) <= 74 || !(datagram.Payload[74] == 18 || datagram.Payload[74] == 19) {
		chargeState = types.EmCurrentState(datagram.Payload[1])
	} else {
		chargeState = types.EmCurrentState(datagram.Payload[74])
	}
	if emproto4go.CompareAndSet(&charge.ChargeState_, chargeState) {
		changed = true
	}
	if emproto4go.CompareAndSet(&charge.ChargeId_, types.ChargeId(emproto4go.ReadString(datagram.Payload[2:18]))) {
		changed = true
	}
	if emproto4go.CompareAndSet(&charge.StartType_, datagram.Payload[18]) {
		changed = true
	}
	if emproto4go.CompareAndSet(&charge.ChargeType_, datagram.Payload[19]) {
		changed = true
	}
	if emproto4go.CompareAndSet(&charge.MaxDuration_, emproto4go.ReadDurationMinutes(datagram.Payload, 20)) {
		changed = true
	}
	if emproto4go.CompareAndSet(&charge.MaxEnergy_, emproto4go.ReadEnergy16(datagram.Payload, 22)) {
		changed = true
	}
	if emproto4go.CompareAndSet(&charge.ReservationTime_, emproto4go.ReadTimestamp(datagram.Payload, 26)) {
		changed = true
	}
	if emproto4go.CompareAndSet(&charge.UserId_, types.UserId(emproto4go.ReadString(datagram.Payload[30:46]))) {
		changed = true
	}
	if emproto4go.CompareAndSet(&charge.MaxCurrent_, types.Amps(datagram.Payload[46])) {
		changed = true
	}
	if emproto4go.CompareAndSet(&charge.StartTime_, emproto4go.ReadTimestamp(datagram.Payload, 47)) {
		changed = true
	}
	if emproto4go.CompareAndSet(&charge.Duration_, emproto4go.ReadDurationSeconds(datagram.Payload, 51)) {
		changed = true
	}
	if emproto4go.CompareAndSet(&charge.StartEnergyCounter_, *emproto4go.ReadEnergy32(datagram.Payload, 55)) {
		changed = true
	}
	if emproto4go.CompareAndSet(&charge.CurrentEnergyCounter_, *emproto4go.ReadEnergy32(datagram.Payload, 59)) {
		changed = true
	}
	if emproto4go.CompareAndSet(&charge.ChargedEnergy_, *emproto4go.ReadEnergy32(datagram.Payload, 63)) {
		changed = true
	}
	if emproto4go.CompareAndSet(&charge.ChargePrice_, float32(binary.BigEndian.Uint32(datagram.Payload[67:71]))*0.01) {
		changed = true
	}
	if emproto4go.CompareAndSet(&charge.FeeType_, datagram.Payload[71]) {
		changed = true
	}
	if emproto4go.CompareAndSet(&charge.ChargeFee_, float32(binary.BigEndian.Uint16(datagram.Payload[72:74]))*0.01) {
		changed = true
	}

	charge.LastFetched = &now

	response := &emproto4go.Datagram{
		Command: itypes.CmdSingleACChargingAck,
		Payload: []byte{0},
	}
	go func() {
		err := evse.SendDatagram(response)
		if err != nil && evse.Communicator().Debug {
			log.Printf("Failed to send SingleACChargingResponse to EVSE %s: %v.", evse.Serial(), err)
		}
	}()

	if changed {
		evse.QueueEvent(types.EvseChargeUpdated)
	}
}

func init() {
	emproto4go.HandlerDelegator.Register(
		SingleAcChargingHandler{},
	)
}
