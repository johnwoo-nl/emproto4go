package handlers

import "github.com/johnwoo-nl/emproto4go/internal"

type ChargeStartStopHandler struct{}

func (h ChargeStartStopHandler) Handles() []internal.EmCommand {
	return []internal.EmCommand{
		internal.CmdChargeStartResponse,
		internal.CmdChargeStopResponse,
	}
}

func (h ChargeStartStopHandler) Handle(*internal.Evse, *internal.Datagram) {
	// Do nothing; this handler is just to stop messages about "No handler for command...".
	// Responses are already handled by Evse.ChargeStart and Evse.ChargeStop methods, and if any
	// other app would start or stop the EVSE, we will pick it up in the SingleAcStatusHandler
	// which will update the state.
}

func init() {
	internal.HandlerDelegator.Register(
		ChargeStartStopHandler{},
	)
}
