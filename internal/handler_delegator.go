package internal

type handlerDelegator struct {
	Handlers []Handler
}

var HandlerDelegator = &handlerDelegator{}

func (handlerDelegator *handlerDelegator) Handle(evse *Evse, datagram *Datagram) uint {
	handlersInvoked := uint(0)
	for _, handler := range handlerDelegator.Handlers {
		for _, cmd := range handler.Handles() {
			if datagram.Command == cmd {
				handlersInvoked++
				handler.Handle(evse, datagram)
				break
			}
		}
	}
	return handlersInvoked
}

func (handlerDelegator *handlerDelegator) Register(handler ...Handler) {
	handlerDelegator.Handlers = append(handlerDelegator.Handlers, handler...)
}
