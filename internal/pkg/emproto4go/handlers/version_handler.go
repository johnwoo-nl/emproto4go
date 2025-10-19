package handlers

import (
	"encoding/binary"
	"time"

	"github.com/johnwoo-nl/emproto4go/internal/pkg/emproto4go"
	"github.com/johnwoo-nl/emproto4go/internal/pkg/itypes"
	"github.com/johnwoo-nl/emproto4go/pkg/types"
)

type VersionHandler struct{}

func (h VersionHandler) Handles() []itypes.EmCommand {
	return []itypes.EmCommand{itypes.CmdGetVersionResponse}
}

func (h VersionHandler) Handle(evse *emproto4go.Evse, datagram *emproto4go.Datagram) {
	if emproto4go.CheckPayloadLength(datagram, evse, 35) {
		return
	}

	changed := false

	info := evse.MutableInfo()
	if emproto4go.CompareAndSet(&info.HardwareVersion_, emproto4go.ReadString(datagram.Payload[0:16])) {
		changed = true
	}
	if emproto4go.CompareAndSet(&info.SoftwareVersion_, emproto4go.ReadString(datagram.Payload[16:32])) {
		changed = true
	}
	if emproto4go.CompareAndSet(&info.Feature_, binary.BigEndian.Uint32(datagram.Payload[32:36])) {
		changed = true
	}
	if len(datagram.Payload) >= 37 {
		if emproto4go.CompareAndSet(&info.SupportNew_, uint32(datagram.Payload[36])) {
			changed = true
		}
	}

	now := time.Now()
	info.LastFetched = &now

	if changed {
		evse.QueueEvent(types.EvseInfoUpdated)
	}
}

func init() {
	emproto4go.HandlerDelegator.Register(
		VersionHandler{},
	)
}
