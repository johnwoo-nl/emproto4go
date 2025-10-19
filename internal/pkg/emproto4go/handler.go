package emproto4go

import (
	"github.com/johnwoo-nl/emproto4go/internal/pkg/itypes"
)

type Handler interface {
	Handles() []itypes.EmCommand
	Handle(evse *Evse, datagram *Datagram)
}
