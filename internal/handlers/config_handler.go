package handlers

import (
	"strings"

	impl "github.com/johnwoo-nl/emproto4go/internal"
	"github.com/johnwoo-nl/emproto4go/types"
)

type ConfigHandler struct{}

func (h ConfigHandler) Handles() []impl.EmCommand {
	return []impl.EmCommand{
		impl.CmdSetAndGetLanguageResponse,
		impl.CmdSetAndGetNameResponse,
		impl.CmdSetAndGetTemperatureUnitResponse,
		impl.CmdSetAndGetOfflineChargeResponse,
		impl.CmdSetAndGetMaxCurrentResponse,
	}
}

func (h ConfigHandler) Handle(evse *impl.Evse, datagram *impl.Datagram) {
	if impl.CheckPayloadLength(datagram, evse, 2) {
		return
	}

	changed := false
	config := evse.MutableConfig()

	switch datagram.Command {
	case impl.CmdSetAndGetLanguageResponse:
		changed = impl.CompareAndSet(&config.Language_, types.EmLanguage(datagram.Payload[1]))
	case impl.CmdSetAndGetNameResponse:
		name := impl.ReadString(datagram.Payload[1:])
		if strings.HasPrefix(name, "ACP#") {
			name = strings.TrimPrefix(name, "ACP#")
		}
		changed = impl.CompareAndSet(&config.Name_, name)
	case impl.CmdSetAndGetTemperatureUnitResponse:
		changed = impl.CompareAndSet(&config.TemperatureUnit_, types.EmTemperatureUnit(datagram.Payload[1]))
	case impl.CmdSetAndGetOfflineChargeResponse:
		changed = impl.CompareAndSet(&config.OfflineCharge_, datagram.Payload[1] == 0)
	case impl.CmdSetAndGetMaxCurrentResponse:
		changed = impl.CompareAndSet(&config.MaxCurrent_, types.Amps(datagram.Payload[1]))
	}

	if changed {
		evse.QueueEvent(types.EvseConfigUpdated)
	}
}

func init() {
	impl.HandlerDelegator.Register(
		ConfigHandler{},
	)
}
