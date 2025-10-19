package handlers

import (
	"strings"

	"github.com/johnwoo-nl/emproto4go/internal/pkg/emproto4go"
	"github.com/johnwoo-nl/emproto4go/internal/pkg/itypes"
	"github.com/johnwoo-nl/emproto4go/pkg/types"
)

type ConfigHandler struct{}

func (h ConfigHandler) Handles() []itypes.EmCommand {
	return []itypes.EmCommand{
		itypes.CmdSetAndGetLanguageResponse,
		itypes.CmdSetAndGetNameResponse,
		itypes.CmdSetAndGetTemperatureUnitResponse,
		itypes.CmdSetAndGetOfflineChargeResponse,
		itypes.CmdSetAndGetMaxCurrentResponse,
	}
}

func (h ConfigHandler) Handle(evse *emproto4go.Evse, datagram *emproto4go.Datagram) {
	if emproto4go.CheckPayloadLength(datagram, evse, 2) {
		return
	}

	changed := false
	config := evse.MutableConfig()

	switch datagram.Command {
	case itypes.CmdSetAndGetLanguageResponse:
		changed = emproto4go.CompareAndSet(&config.Language_, types.EmLanguage(datagram.Payload[1]))
	case itypes.CmdSetAndGetNameResponse:
		name := emproto4go.ReadString(datagram.Payload[1:])
		if strings.HasPrefix(name, "ACP#") {
			name = strings.TrimPrefix(name, "ACP#")
		}
		changed = emproto4go.CompareAndSet(&config.Name_, name)
	case itypes.CmdSetAndGetTemperatureUnitResponse:
		changed = emproto4go.CompareAndSet(&config.TemperatureUnit_, types.EmTemperatureUnit(datagram.Payload[1]))
	case itypes.CmdSetAndGetOfflineChargeResponse:
		changed = emproto4go.CompareAndSet(&config.OfflineCharge_, datagram.Payload[1] == 0)
	case itypes.CmdSetAndGetMaxCurrentResponse:
		changed = emproto4go.CompareAndSet(&config.MaxCurrent_, types.Amps(datagram.Payload[1]))
	}

	if changed {
		evse.QueueEvent(types.EvseConfigUpdated)
	}
}

func init() {
	emproto4go.HandlerDelegator.Register(
		ConfigHandler{},
	)
}
