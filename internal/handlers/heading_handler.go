package handlers

import (
	"log"
	"time"

	impl "github.com/johnwoo-nl/emproto4go/internal"
)

type HeadingHandler struct{}

func (h HeadingHandler) Handles() []impl.EmCommand {
	return []impl.EmCommand{impl.CmdHeading}
}

func (h HeadingHandler) Handle(evse *impl.Evse, _ *impl.Datagram) {
	response := &impl.Datagram{
		Command: impl.CmdHeadingResponse,
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
	impl.HandlerDelegator.Register(
		HeadingHandler{},
	)
}
