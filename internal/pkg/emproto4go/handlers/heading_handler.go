package handlers

import (
	"log"
	"time"

	"github.com/johnwoo-nl/emproto4go/internal/pkg/emproto4go"
	"github.com/johnwoo-nl/emproto4go/internal/pkg/itypes"
)

type HeadingHandler struct{}

func (h HeadingHandler) Handles() []itypes.EmCommand {
	return []itypes.EmCommand{itypes.CmdHeading}
}

func (h HeadingHandler) Handle(evse *emproto4go.Evse, _ *emproto4go.Datagram) {
	response := &emproto4go.Datagram{
		Command: itypes.CmdHeadingResponse,
		Payload: []byte{0},
	}
	go func() {
		err := evse.SendDatagram(response)
		if err == nil {
			now := time.Now()
			evse.LastActiveLogin = &now
		} else if evse.Communicator().Debug {
			log.Printf("Failed to send HeadingResponse to EVSE %s: %v. This may result in the login session expiring and the communicator running the Login flow again.", evse.Serial(), err)
		}
	}()
}

func init() {
	emproto4go.HandlerDelegator.Register(
		HeadingHandler{},
	)
}
