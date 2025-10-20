package handlers

import (
	"encoding/binary"
	"slices"

	impl "github.com/johnwoo-nl/emproto4go/internal"
	"github.com/johnwoo-nl/emproto4go/types"
)

type LoginInfoHandler struct{}

func (h LoginInfoHandler) Handles() []impl.EmCommand {
	return []impl.EmCommand{impl.CmdLogin, impl.CmdLoginResponse}
}

func (h LoginInfoHandler) Handle(evse *impl.Evse, datagram *impl.Datagram) {
	if impl.CheckPayloadLength(datagram, evse, 54) {
		return
	}

	changed := false
	info := evse.MutableInfo()

	if impl.CompareAndSet(&info.EvseType_, datagram.Payload[0]) {
		changed = true
	}

	brand := impl.ReadString(datagram.Payload[1:17])
	model := impl.ReadString(datagram.Payload[17:33])
	if len(datagram.Payload) >= 151 {
		brand += impl.ReadString(datagram.Payload[119:135])
		model += impl.ReadString(datagram.Payload[135:151])
	}
	if impl.CompareAndSet(&info.Brand_, brand) {
		changed = true
	}
	if impl.CompareAndSet(&info.Model_, model) {
		changed = true
	}

	if impl.CompareAndSet(&info.HardwareVersion_, impl.ReadString(datagram.Payload[33:49])) {
		changed = true
	}
	if impl.CompareAndSet(&info.MaxPower_, types.Watts(binary.BigEndian.Uint32(datagram.Payload[49:53]))) {
		changed = true
	}
	if impl.CompareAndSet(&info.MaxCurrent_, types.Amps(datagram.Payload[53])) {
		changed = true
	}

	phases := types.Phases1p
	if slices.Contains([]byte{10, 11, 12, 13, 14, 15, 22, 23, 24, 25}, info.EvseType()) {
		phases = types.Phases3p
	}
	if impl.CompareAndSet(&info.Phases_, phases) {
		changed = true
	}

	byte70 := byte(0)
	if len(datagram.Payload) >= 119 && slices.Contains([]byte{10, 11, 12, 13, 14, 15, 22, 23, 24, 25}, info.EvseType()) {
		byte70 = datagram.Payload[70]
	}
	if impl.CompareAndSet[byte](&info.Byte70_, byte70) {
		changed = true
	}

	if changed {
		evse.QueueEvent(types.EvseInfoUpdated)
	}
}

func init() {
	impl.HandlerDelegator.Register(
		LoginInfoHandler{},
	)
}
