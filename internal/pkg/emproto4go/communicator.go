package emproto4go

import (
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/johnwoo-nl/emproto4go/internal/pkg/itypes"
	"github.com/johnwoo-nl/emproto4go/pkg/types"
)

type Communicator struct {
	AppName_ types.UserId
	Debug    bool
	started  bool

	evses      map[types.EmSerial]*Evse
	evsesMutex sync.RWMutex

	udpConn      *net.UDPConn
	udpConnMutex sync.Mutex

	watchers      []EventWatcher
	watchersMutex sync.RWMutex

	debouncedEvents      map[string]types.EmEvent
	debouncedEventsMutex sync.Mutex
	debounceTimers       map[string]*time.Timer
	debounceTimersMutex  sync.Mutex

	tickerStopChan chan struct{}

	tickerRunning bool
}

func CreateCommunicator(appName types.UserId, debug bool) *Communicator {
	return &Communicator{
		AppName_:        appName,
		Debug:           debug,
		evses:           make(map[types.EmSerial]*Evse),
		debouncedEvents: make(map[string]types.EmEvent),
		debounceTimers:  make(map[string]*time.Timer),
	}
}

func (communicator *Communicator) AppName() types.UserId {
	return communicator.AppName_
}

func (communicator *Communicator) GetEvse(serial types.EmSerial) types.EmEvse {
	return communicator.getEvseImpl(serial)
}

func (communicator *Communicator) getEvseImpl(serial types.EmSerial) *Evse {
	communicator.evsesMutex.RLock()
	defer communicator.evsesMutex.RUnlock()

	return communicator.evses[serial]
}

func (communicator *Communicator) GetEvses() []types.EmEvse {
	communicator.evsesMutex.RLock()
	defer communicator.evsesMutex.RUnlock()

	evses := make([]types.EmEvse, 0, len(communicator.evses))
	for _, evse := range communicator.evses {
		evses = append(evses, evse)
	}
	return evses
}

func (communicator *Communicator) DefineEvse(serial types.EmSerial) types.EmEvse {
	return communicator.defineEvseImpl(serial)
}

func (communicator *Communicator) defineEvseImpl(serial types.EmSerial) *Evse {
	existing := communicator.getEvseImpl(serial)
	if existing != nil {
		return existing
	}

	evse := Evse{
		communicator: communicator,
		info:         &EvseInfo{serial: serial},
		state:        &EvseState{},
		charge:       &EvseCharge{},
		config:       &EvseConfig{},
	}
	evse.info.evse = &evse
	evse.charge.evse = &evse
	evse.config.evse = &evse

	communicator.evsesMutex.Lock()
	defer communicator.evsesMutex.Unlock()

	communicator.evses[serial] = &evse
	communicator.QueueEvent(&evse, types.EvseAdded)
	return &evse
}

func (communicator *Communicator) RemoveEvse(evse types.EmEvse) error {
	// Cannot remove an EVSE that is online, since it would be re-discovered within a few seconds.
	if evse.IsOnline() {
		return types.EvseOnlineError{Evse: evse}
	}

	communicator.evsesMutex.Lock()
	defer communicator.evsesMutex.Unlock()

	// Clear any queued events for this EVSE and add one single EvseRemoved event.
	communicator.clearQueuedEvents(evse)
	communicator.QueueEvent(evse.(*Evse), types.EvseRemoved)

	delete(communicator.evses, evse.Serial())
	return nil
}

func (communicator *Communicator) Tick() {
	communicator.evsesMutex.RLock()
	var evsesCopy []*Evse
	for _, evse := range communicator.evses {
		evsesCopy = append(evsesCopy, evse)
	}
	communicator.evsesMutex.RUnlock()

	for _, evse := range evsesCopy {
		evse.Tick()
	}
}

func (communicator *Communicator) Start() error {
	addr := net.UDPAddr{
		Port: 28376,
		IP:   net.ParseIP("0.0.0.0"),
	}
	conn, err := net.ListenUDP("udp", &addr)
	if err != nil {
		return err
	}
	communicator.udpConnMutex.Lock()
	communicator.udpConn = conn
	communicator.started = true
	communicator.udpConnMutex.Unlock()

	if communicator.Debug {
		log.Printf("Communicator started, UDP listener on port %v", addr.Port)
	}

	// Start ticker goroutine if not already running
	if !communicator.tickerRunning {
		communicator.tickerStopChan = make(chan struct{})
		communicator.tickerRunning = true
		go func() {
			ticker := time.NewTicker(5 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-communicator.tickerStopChan:
					return
				case <-ticker.C:
					if communicator.started {
						communicator.Tick()
					}
				}
			}
		}()
	}

	// UDP receiver loop.
	go func() {
		defer communicator.stopped()
		buf := make([]byte, 4096)
		for {
			if !communicator.started {
				return
			}
			n, remoteAddr, err := conn.ReadFrom(buf)
			if n > 0 {
				communicator.packetReceived(buf[:n], &remoteAddr)
			} else if err != nil {
				if !strings.Contains(err.Error(), "use of closed network connection") {
					log.Printf("UDP read error: %v", err)
				}
				return
			}
		}
	}()

	return nil
}

func (communicator *Communicator) Stop() {
	if !communicator.started {
		return
	}
	communicator.udpConnMutex.Lock()
	communicator.started = false
	communicator.udpConnMutex.Unlock()

	// Stop ticker goroutine
	if communicator.tickerRunning && communicator.tickerStopChan != nil {
		close(communicator.tickerStopChan)
		communicator.tickerRunning = false
	}
	communicator.stopped()
}

func (communicator *Communicator) stopped() {
	communicator.udpConnMutex.Lock()
	if communicator.udpConn != nil {
		_ = communicator.udpConn.Close()
		communicator.udpConn = nil
	}
	communicator.udpConnMutex.Unlock()

	if communicator.started {
		if communicator.Debug {
			log.Printf("Communicator stopped unexpectedly; will restart")
		}
		// If it was stopped unexpectedly, try to restart after a delay. Don't send OFFLINE events, because we may
		// restart quickly (time-based OFFLINE events could anyway be sent if restart takes too long).
		communicator.restart()
	} else {
		if communicator.Debug {
			log.Printf("Communicator stopped")
		}
		// If we were actually asked to stop, then send OFFLINE events for all EVSEs that were online, and clear
		// the LastSeen timestamps so we can get ONLINE events again if we'd be started again quickly.
		communicator.evsesMutex.RLock()
		for _, evse := range communicator.evses {
			communicator.clearQueuedEvents(evse)
			if evse.IsLoggedIn() {
				communicator.QueueEvent(evse, types.EvseLoggedOut)
				evse.LastActiveLogin = nil
			}
			if evse.IsOnline() {
				communicator.QueueEvent(evse, types.EvseOffline)
				evse.LastSeen = nil
			}
		}
	}
}

func (communicator *Communicator) restart() {
	if !communicator.started {
		return
	}
	err := communicator.Start()
	if err != nil {
		log.Printf("Failed to restart communicator (will retry): %v", err)
		go func() {
			time.Sleep(10 * time.Second)
			communicator.restart()
		}()
		return
	}
}

func (communicator *Communicator) packetReceived(data []byte, addr *net.Addr) {
	datagram, err := Decode(data)

	if err != nil {
		log.Printf("Failed to parse datagram from %v: %v\nDatagram bytes: %v", addr, err, data)
		return
	}
	if datagram == nil {
		// Not an EvseMaster datagram.
		return
	}

	// DefineEvse will return an existing instance if it exists by serial.
	evse := communicator.defineEvseImpl(datagram.Serial)
	evse.DatagramReceived(datagram, addr)
}

func (communicator *Communicator) SendDatagram(evse *Evse, datagram *Datagram) error {
	if !evse.IsOnline() {
		return types.EvseOfflineError{Evse: evse}
	}
	if !evse.IsLoggedIn() && datagram.Command != itypes.CmdRequestLogin && datagram.Command != itypes.CmdLoginConfirm {
		return types.EvseNotLoggedInError{Evse: evse}
	}

	datagram.Serial = evse.Serial()
	if (datagram.Password == "") && (evse.password != "") {
		datagram.Password = evse.password
	}

	data, err := datagram.Encode()
	if err != nil {
		return err
	}
	communicator.udpConnMutex.Lock()
	defer communicator.udpConnMutex.Unlock()

	if communicator.udpConn == nil {
		return types.EvseOfflineError{Evse: evse}
	}
	addr := &net.UDPAddr{IP: evse.IP(), Port: evse.Port()}
	if evse.communicator.Debug {
		log.Printf("-> SEND %+v to %s", datagram, addr)
	}
	n, err := communicator.udpConn.WriteTo(data, addr)
	if err != nil {
		return err
	}
	if n != len(data) {
		return itypes.DatagramSendError{
			Evse:    evse,
			Command: datagram.Command,
			Message: fmt.Sprintf("incomplete write (%d of %d bytes sent)", n, len(data)),
		}
	}
	return nil
}

func (communicator *Communicator) Watch(evse types.EmEvse, eventTypes []types.EmEventType, channel chan<- types.EmEvent) types.EmEventWatcher {
	if evse == nil {
		return communicator.WatchImpl(nil, eventTypes, channel)
	} else {
		evseInstance := communicator.getEvseImpl(evse.Serial())
		return communicator.WatchImpl(evseInstance, eventTypes, channel)
	}
}

func (communicator *Communicator) WatchImpl(evse *Evse, eventTypes []types.EmEventType, channel chan<- types.EmEvent) *EventWatcher {
	watcher := &EventWatcher{
		communicator: communicator,
		evse:         evse,
		eventTypes:   eventTypes,
		channel:      channel,
	}
	communicator.watchersMutex.Lock()
	defer communicator.watchersMutex.Unlock()
	communicator.watchers = append(communicator.watchers, *watcher)
	return watcher
}

func (communicator *Communicator) RemoveWatcher(watcher *EventWatcher) {
	communicator.watchersMutex.Lock()
	defer communicator.watchersMutex.Unlock()

	for i, w := range communicator.watchers {
		if &w == watcher {
			// Remove watcher from slice
			communicator.watchers = append(communicator.watchers[:i], communicator.watchers[i+1:]...)
			return
		}
	}
}

func (communicator *Communicator) QueueEvent(evse *Evse, eventType types.EmEventType) {
	if evse == nil {
		return
	}

	key := string(evse.Serial()) + ":" + string(eventType)
	eventInstance := types.EmEvent{Evse: evse, Type: eventType, Timestamp: time.Now()}

	communicator.debouncedEventsMutex.Lock()
	communicator.debouncedEvents[key] = eventInstance
	communicator.debouncedEventsMutex.Unlock()

	communicator.debounceTimersMutex.Lock()
	timer, exists := communicator.debounceTimers[key]
	if exists {
		// If timer exists, check if eventType has been in queue for >2000ms
		communicator.debouncedEventsMutex.Lock()
		firstQueued := communicator.debouncedEvents[key].Timestamp
		communicator.debouncedEventsMutex.Unlock()
		elapsed := time.Since(firstQueued)
		if elapsed >= 2000*time.Millisecond {
			timer.Stop()
			delete(communicator.debounceTimers, key)
			communicator.debounceTimersMutex.Unlock()
			communicator.dispatchDebouncedEvent(key)
			return
		} else {
			timer.Stop()
		}
	}
	// Start a new timer for 400ms debounce
	timer = time.AfterFunc(400*time.Millisecond, func() {
		communicator.dispatchDebouncedEvent(key)
	})
	communicator.debounceTimers[key] = timer
	communicator.debounceTimersMutex.Unlock()

	// - Online/Offline events will also result in an InfoUpdated event due to IsOnline changing;
	// - LoggedIn/LoggedOut events will also result in an InfoUpdated event due to IsLoggedIn changing;
	if eventType == types.EvseOnline || eventType == types.EvseOffline || eventType == types.EvseLoggedIn || eventType == types.EvseLoggedOut {
		evse.communicator.QueueEvent(evse, types.EvseInfoUpdated)
	}
	// - ChargeStarted/ChargeStopped events will also result in a StateUpdated event due to CurrentState changing.
	if eventType == types.EvseChargeStarted || eventType == types.EvseChargeStopped {
		evse.communicator.QueueEvent(evse, types.EvseStateUpdated)
	}
}

func (communicator *Communicator) dispatchDebouncedEvent(key string) {
	communicator.debouncedEventsMutex.Lock()
	event, exists := communicator.debouncedEvents[key]
	if !exists {
		communicator.debouncedEventsMutex.Unlock()
		return
	}
	delete(communicator.debouncedEvents, key)
	communicator.debouncedEventsMutex.Unlock()

	communicator.debounceTimersMutex.Lock()
	timer, exists := communicator.debounceTimers[key]
	if exists {
		timer.Stop()
		delete(communicator.debounceTimers, key)
	}
	communicator.debounceTimersMutex.Unlock()

	communicator.dispatchEvent(event)
}

func (communicator *Communicator) dispatchEvent(event types.EmEvent) {
	communicator.watchersMutex.RLock()
	defer communicator.watchersMutex.RUnlock()

	for _, watcher := range communicator.watchers {
		watcher.Notify(event)
	}
}

func (communicator *Communicator) clearQueuedEvents(evse types.EmEvse) {
	serialPrefix := string(evse.Serial()) + ":"

	// Remove debounced events
	communicator.debouncedEventsMutex.Lock()
	for key := range communicator.debouncedEvents {
		if strings.HasPrefix(key, serialPrefix) {
			delete(communicator.debouncedEvents, key)
		}
	}
	communicator.debouncedEventsMutex.Unlock()

	// Remove and stop debounce timers
	communicator.debounceTimersMutex.Lock()
	for key, timer := range communicator.debounceTimers {
		if strings.HasPrefix(key, serialPrefix) {
			timer.Stop()
			delete(communicator.debounceTimers, key)
		}
	}
	communicator.debounceTimersMutex.Unlock()
}
