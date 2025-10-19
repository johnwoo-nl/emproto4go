package handlers

import (
	"encoding/binary"
	"slices"

	"github.com/johnwoo-nl/emproto4go/internal/pkg/emproto4go"
	"github.com/johnwoo-nl/emproto4go/internal/pkg/itypes"
	"github.com/johnwoo-nl/emproto4go/pkg/types"
)

type LoginInfoHandler struct{}

func (h LoginInfoHandler) Handles() []itypes.EmCommand {
	return []itypes.EmCommand{itypes.CmdLogin, itypes.CmdLoginResponse}
}

func (h LoginInfoHandler) Handle(evse *emproto4go.Evse, datagram *emproto4go.Datagram) {
	if emproto4go.CheckPayloadLength(datagram, evse, 54) {
		return
	}

	changed := false
	info := evse.MutableInfo()

	if emproto4go.CompareAndSet(&info.EvseType_, datagram.Payload[0]) {
		changed = true
	}

	brand := emproto4go.ReadString(datagram.Payload[1:17])
	model := emproto4go.ReadString(datagram.Payload[17:33])
	if len(datagram.Payload) >= 151 {
		brand += emproto4go.ReadString(datagram.Payload[119:135])
		model += emproto4go.ReadString(datagram.Payload[135:151])
	}
	if emproto4go.CompareAndSet(&info.Brand_, brand) {
		changed = true
	}
	if emproto4go.CompareAndSet(&info.Model_, model) {
		changed = true
	}

	if emproto4go.CompareAndSet(&info.HardwareVersion_, emproto4go.ReadString(datagram.Payload[33:49])) {
		changed = true
	}
	if emproto4go.CompareAndSet(&info.MaxPower_, types.Watts(binary.BigEndian.Uint32(datagram.Payload[49:53]))) {
		changed = true
	}
	if emproto4go.CompareAndSet(&info.MaxCurrent_, types.Amps(datagram.Payload[53])) {
		changed = true
	}

	phases := types.Phases1p
	if slices.Contains([]byte{10, 11, 12, 13, 14, 15, 22, 23, 24, 25}, info.EvseType()) {
		phases = types.Phases3p
	}
	if emproto4go.CompareAndSet(&info.Phases_, phases) {
		changed = true
	}

	byte70 := byte(0)
	if len(datagram.Payload) >= 119 && slices.Contains([]byte{10, 11, 12, 13, 14, 15, 22, 23, 24, 25}, info.EvseType()) {
		byte70 = datagram.Payload[70]
	}
	if emproto4go.CompareAndSet[byte](&info.Byte70_, byte70) {
		changed = true
	}

	if changed {
		evse.QueueEvent(types.EvseInfoUpdated)
	}
}

func init() {
	emproto4go.HandlerDelegator.Register(
		LoginInfoHandler{},
	)
}
