package handlers

import (
	"github.com/johnwoo-nl/emproto4go/internal/pkg/emproto4go"
	"github.com/johnwoo-nl/emproto4go/internal/pkg/itypes"
)

type ChargeStartStopHandler struct{}

func (h ChargeStartStopHandler) Handles() []itypes.EmCommand {
	return []itypes.EmCommand{
		itypes.CmdChargeStartResponse,
		itypes.CmdChargeStopResponse,
	}
}

func (h ChargeStartStopHandler) Handle(*emproto4go.Evse, *emproto4go.Datagram) {
	// Do nothing; this handler is just to stop messages about "No handler for command...".
	// Responses are already handled by Evse.ChargeStart and Evse.ChargeStop methods, and if any
	// other app would start or stop the EVSE, we will pick it up in the SingleAcStatusHandler
	// which will update the state.
}

func init() {
	emproto4go.HandlerDelegator.Register(
		ChargeStartStopHandler{},
	)
}
