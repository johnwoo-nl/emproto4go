package handlers

import (
	"encoding/binary"
	"log"
	"math"

	"github.com/johnwoo-nl/emproto4go/internal/pkg/emproto4go"
	"github.com/johnwoo-nl/emproto4go/internal/pkg/itypes"
	"github.com/johnwoo-nl/emproto4go/pkg/types"
)

type SingleAcStatusHandler struct{}

func (h SingleAcStatusHandler) Handles() []itypes.EmCommand {
	return []itypes.EmCommand{itypes.CmdSingleACStatus}
}

func (h SingleAcStatusHandler) Handle(evse *emproto4go.Evse, datagram *emproto4go.Datagram) {
	if emproto4go.CheckPayloadLength(datagram, evse, 25) {
		return
	}

	state := evse.MutableState()
	oldMetaState := evse.MetaState()
	changed := false

	if emproto4go.CompareAndSet(&state.LineId_, types.LineId(datagram.Payload[0])) {
		changed = true
	}

	// L1
	if emproto4go.CompareAndSet(&state.L1Voltage_, types.Volts(float32(binary.BigEndian.Uint16(datagram.Payload[1:3]))*0.1)) {
		changed = true
	}
	if emproto4go.CompareAndSet(&state.L1Current_, types.Amps(float32(binary.BigEndian.Uint16(datagram.Payload[3:5]))*0.01)) {
		changed = true
	}

	// L2 and L3, only if datagram payload has it.
	l2Voltage := types.Volts(0.0)
	l2Current := types.Amps(0.0)
	l3Voltage := types.Volts(0.0)
	l3Current := types.Amps(0.0)
	if len(datagram.Payload) >= 33 {
		l2Voltage = types.Volts(float32(binary.BigEndian.Uint16(datagram.Payload[25:27])) * 0.1)
		l2Current = types.Amps(float32(binary.BigEndian.Uint16(datagram.Payload[27:29])) * 0.01)
		l3Voltage = types.Volts(float32(binary.BigEndian.Uint16(datagram.Payload[29:31])) * 0.1)
		l3Current = types.Amps(float32(binary.BigEndian.Uint16(datagram.Payload[31:33])) * 0.01)
	}
	if emproto4go.CompareAndSet(&state.L2Voltage_, l2Voltage) {
		changed = true
	}
	if emproto4go.CompareAndSet(&state.L2Current_, l2Current) {
		changed = true
	}
	if emproto4go.CompareAndSet(&state.L3Voltage_, l3Voltage) {
		changed = true
	}
	if emproto4go.CompareAndSet(&state.L3Current_, l3Current) {
		changed = true
	}

	// Total power.
	currentPower := types.Watts(binary.BigEndian.Uint32(datagram.Payload[5:9]))
	computedPower := float64(state.L1Current_)*float64(state.L1Voltage_) +
		float64(state.L2Current_)*float64(state.L2Voltage_) +
		float64(state.L3Current_)*float64(state.L3Voltage_)
	currentPower = types.Watts(math.Max(float64(currentPower), computedPower))
	if emproto4go.CompareAndSet(&state.CurrentPower_, currentPower) {
		changed = true
	}

	if emproto4go.CompareAndSet(&state.EnergyCounter_, types.KWh(float64(binary.BigEndian.Uint32(datagram.Payload[9:13]))*0.01)) {
		changed = true
	}

	// Temperatures
	if emproto4go.CompareAndSet(&state.InnerTemp_, emproto4go.ReadTemperature(datagram.Payload, 13)) {
		changed = true
	}
	if emproto4go.CompareAndSet(&state.OuterTemp_, emproto4go.ReadTemperature(datagram.Payload, 15)) {
		changed = true
	}

	if emproto4go.CompareAndSet(&state.EmergencyBtnState_, types.EmEmergencyBtnState(datagram.Payload[17])) {
		changed = true
	}

	if emproto4go.CompareAndSet(&state.GunState_, types.EmGunState(datagram.Payload[18])) {
		changed = true
	}
	if emproto4go.CompareAndSet(&state.OutputState_, types.EmOutputState(datagram.Payload[19])) {
		changed = true
	}

	if emproto4go.CompareAndSet(&state.NewProtocol_, len(datagram.Payload) > 33) {
		changed = true
	}

	currentState := types.EmCurrentState(datagram.Payload[20])
	if state.NewProtocol_ {
		var byte34 = datagram.Payload[34]
		if byte34 == 18 || byte34 == 19 {
			currentState = types.EmCurrentState(byte34)
		}
	}
	if emproto4go.CompareAndSet(&state.CurrentState_, currentState) {
		changed = true
	}

	errors := ParseErrors(binary.BigEndian.Uint32(datagram.Payload[21:25]))
	if !emproto4go.SliceEqual(state.Errors_, errors) {
		state.Errors_ = errors
		changed = true
	}

	response := &emproto4go.Datagram{
		Command: itypes.CmdSingleACStatusAck,
		Payload: []byte{1},
	}
	go func() {
		err := evse.SendDatagram(response)
		if err != nil && evse.Communicator().Debug {
			log.Printf("Failed to send SingleACStatusResponse to EVSE %s: %v.", evse.Serial(), err)
		}
	}()

	if changed {
		evse.QueueEvent(types.EvseStateUpdated)
	}
	newMetaState := evse.MetaState()
	if oldMetaState != types.MetaStateCharging && newMetaState == types.MetaStateCharging {
		evse.QueueEvent(types.EvseChargeStarted)
	} else if oldMetaState == types.MetaStateCharging && newMetaState != types.MetaStateCharging {
		evse.QueueEvent(types.EvseChargeStopped)
	}
}

func ParseErrors(data uint32) []types.EmError {
	var errors []types.EmError
	for i := 0; i < 32; i++ {
		if (data & (1 << i)) != 0 {
			errors = append(errors, types.EmError(i))
		}
	}
	return errors
}

func init() {
	emproto4go.HandlerDelegator.Register(
		SingleAcStatusHandler{},
	)
}
