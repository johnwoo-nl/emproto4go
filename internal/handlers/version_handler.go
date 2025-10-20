package handlers

import (
	"encoding/binary"
	"time"

	impl "github.com/johnwoo-nl/emproto4go/internal"
	"github.com/johnwoo-nl/emproto4go/types"
)

type VersionHandler struct{}

func (h VersionHandler) Handles() []impl.EmCommand {
	return []impl.EmCommand{impl.CmdGetVersionResponse}
}

func (h VersionHandler) Handle(evse *impl.Evse, datagram *impl.Datagram) {
	if impl.CheckPayloadLength(datagram, evse, 35) {
		return
	}

	changed := false

	info := evse.MutableInfo()
	if impl.CompareAndSet(&info.HardwareVersion_, impl.ReadString(datagram.Payload[0:16])) {
		changed = true
	}
	if impl.CompareAndSet(&info.SoftwareVersion_, impl.ReadString(datagram.Payload[16:32])) {
		changed = true
	}
	if impl.CompareAndSet(&info.Feature_, binary.BigEndian.Uint32(datagram.Payload[32:36])) {
		changed = true
	}
	if len(datagram.Payload) >= 37 {
		if impl.CompareAndSet(&info.SupportNew_, uint32(datagram.Payload[36])) {
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
	impl.HandlerDelegator.Register(
		VersionHandler{},
	)
}
