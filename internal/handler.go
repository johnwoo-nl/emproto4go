package internal

type Handler interface {
	Handles() []EmCommand
	Handle(evse *Evse, datagram *Datagram)
}
