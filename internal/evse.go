package internal

import (
	"encoding/binary"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/johnwoo-nl/emproto4go/types"
)

type Evse struct {
	communicator *Communicator
	info         *EvseInfo
	state        *EvseState
	charge       *EvseCharge
	config       *EvseConfig

	ip              net.IP
	port            int
	LastSeen        *time.Time
	LastActiveLogin *time.Time
	password        types.EmPassword

	waitersMutex sync.Mutex
	waiters      map[EmCommand][]chan *Datagram
}

func (evse *Evse) Communicator() *Communicator {
	return evse.communicator
}

func (evse *Evse) IsOnline() bool {
	if evse.LastSeen == nil {
		return false
	}
	return evse.LastSeen.After(time.Now().Add(-11 * time.Second))
}

func (evse *Evse) IsLoggedIn() bool {
	if !evse.IsOnline() || evse.LastActiveLogin == nil {
		return false
	}
	return evse.LastActiveLogin.After(time.Now().Add(-15 * time.Second))
}

func (evse *Evse) MetaState() types.EmMetaState {
	if !evse.IsOnline() {
		return types.MetaStateOffline
	} else if !evse.IsLoggedIn() {
		return types.MetaStateNotLoggedIn
	} else if len(evse.State().Errors()) > 0 {
		return types.MetaStateError
	} else if evse.State().OutputState() == types.OutputStateCharging {
		return types.MetaStateCharging
	} else if evse.State().GunState() > types.GunNotConnected {
		return types.MetaStatePluggedIn
	} else {
		return types.MetaStateIdle
	}
}

// Info implements types.EmEvse interface.
func (evse *Evse) Info() types.EmEvseInfo {
	return evse.info
}

// MutableInfo not exposed to library users, only internally for updating.
func (evse *Evse) MutableInfo() *EvseInfo {
	return evse.info
}

// State implements types.EmEvse interface.
func (evse *Evse) State() types.EmEvseState {
	return evse.state
}

// MutableState not exposed to library users, only internally for updating.
func (evse *Evse) MutableState() *EvseState {
	return evse.state
}

// Charge implements types.EmEvse interface.
func (evse *Evse) Charge() types.EmEvseCharge {
	return evse.charge
}

// MutableCharge not exposed to library users, only internally for updating.
func (evse *Evse) MutableCharge() *EvseCharge {
	return evse.charge
}

// Config implements types.EmEvse interface.
func (evse *Evse) Config() types.EmEvseConfig {
	return evse.config
}

// MutableConfig not exposed to library users, only internally for updating.
func (evse *Evse) MutableConfig() *EvseConfig {
	return evse.config
}

func (evse *Evse) Serial() types.EmSerial {
	return evse.info.Serial()
}

func (evse *Evse) Label() string {
	if evse.config.Name_ != "" {
		return evse.config.Name_
	}
	if evse.info.Brand_ != "" && evse.info.Model_ != "" {
		return fmt.Sprintf("%s %s", evse.info.Brand_, evse.info.Model_)
	}
	return string(evse.info.Serial())
}

func (evse *Evse) IP() net.IP {
	return evse.ip
}

func (evse *Evse) Port() int {
	return evse.port
}

func (evse *Evse) UsePassword(password types.EmPassword) error {
	// If not online, just store the password for later use.
	if !evse.IsOnline() {
		evse.password = password
		return nil
	}

	// If online, attempt to (re)log in immediately. This returns an error if the password is invalid,
	// or updates the EVSE password if login succeeds.
	return evse.Login(password)
}

func (evse *Evse) StartCharge(params types.ChargeStartParams) (types.ChargeStartResult, error) {
	if !evse.IsOnline() {
		return types.ChargeStartResult{
			ErrorReason:  types.ChargeStartErrorEvseOffline,
			ErrorMessage: GetChargeStartErrorReasonMessage(types.ChargeStartErrorEvseOffline),
		}, types.EvseOfflineError{Evse: evse}
	}
	if !evse.IsLoggedIn() {
		return types.ChargeStartResult{
			ErrorReason:  types.ChargeStartErrorEvseNotLoggedIn,
			ErrorMessage: GetChargeStartErrorReasonMessage(types.ChargeStartErrorEvseNotLoggedIn),
		}, types.EvseNotLoggedInError{Evse: evse}
	}

	startChargeDatagram := evse.createChargeStartDatagram(params)
	sendErr := evse.SendDatagram(startChargeDatagram)
	if sendErr != nil {
		return types.ChargeStartResult{
			ErrorReason:  types.ChargeStartErrorSendFailed,
			ErrorMessage: GetChargeStartErrorReasonMessage(types.ChargeStartErrorSendFailed),
		}, sendErr
	}
	response, recvErr := evse.WaitForDatagram(5*time.Second, CmdChargeStartResponse)
	if recvErr != nil {
		return types.ChargeStartResult{
			ErrorReason:  types.ChargeStartErrorNoConfirmation,
			ErrorMessage: GetChargeStartErrorReasonMessage(types.ChargeStartErrorNoConfirmation),
		}, recvErr
	}

	if CheckPayloadLength(response, evse, 5) {
		return types.ChargeStartResult{
			ErrorReason:  types.ChargeStartErrorNoConfirmation,
			ErrorMessage: GetChargeStartErrorReasonMessage(types.ChargeStartErrorNoConfirmation),
		}, types.EvseInvalidDatagramError{Evse: evse, ResponseCommand: uint16(response.Command)}
	}

	errorReason := types.ChargeStartErrorReason(response.Payload[3])
	errorMessage := GetChargeStartErrorReasonMessage(errorReason)
	if errorReason != types.ChargeStartOK {
		return types.ChargeStartResult{ErrorReason: errorReason, ErrorMessage: errorMessage},
			types.EvseChargeStartError{Evse: evse, ErrorReason: errorReason, ErrorMessage: errorMessage}
	}

	return types.ChargeStartResult{
		ErrorReason: types.ChargeStartOK,
		LineId:      response.Payload[0],
		Current:     types.Amps(response.Payload[4]),
	}, nil
}

func (evse *Evse) createChargeStartDatagram(params types.ChargeStartParams) *Datagram {
	now := time.Now()

	lineId := 2
	chargeType := 1
	if params.ForceSinglePhase {
		if evse.info.CanForceSinglePhase() {
			lineId = 1
			chargeType = 11
		} else {
			evse.communicator.Logger_.Warnf("[emproto4go] ChargeStart for EVSE %s requested ForceSinglePhase, but this EVSE does not support it. Will use all available phases.", evse.Serial())
		}
	}
	maxCurrent := min(params.MaxCurrent, evse.config.MaxCurrent())
	if maxCurrent <= 6 {
		maxCurrent = 6
	} else if maxCurrent > evse.info.MaxCurrent() {
		maxCurrent = evse.info.MaxCurrent()
	}
	userId := params.UserId
	if userId == "" {
		userId = evse.communicator.AppName_
	}
	chargeId := MakeChargeId(params.ChargeId)
	startAt := now
	isReservation := 0
	if params.StartAt.Unix() > 0 {
		startAt = params.StartAt
		if startAt.After(now.Add(5 * time.Second)) {
			isReservation = 1
		}
	}

	var payload [47]byte
	payload[0] = byte(lineId)
	WriteUserId(payload[1:17], userId)
	copy(payload[17:33], chargeId)
	payload[33] = byte(isReservation)
	binary.BigEndian.PutUint32(payload[34:38], TimeToEmTimestamp(&startAt))
	payload[38] = byte(1) // startType, always 1
	payload[39] = byte(chargeType)
	if params.MaxDuration <= 0 {
		// Zero/uninitialized, means no limit.
		binary.BigEndian.PutUint16(payload[40:42], 0xFFFF)
	} else {
		binary.BigEndian.PutUint16(payload[40:42], max(1, uint16(params.MaxDuration.Minutes())))
	}
	if params.MaxEnergy <= 0 {
		binary.BigEndian.PutUint16(payload[42:44], 0xFFFF)
	} else {
		binary.BigEndian.PutUint16(payload[42:44], uint16(params.MaxEnergy*100))
	}
	binary.BigEndian.PutUint16(payload[44:46], 0xFFFF) // Param3, meaning unknown, always 0xFFFF in the OEM app. Leaving it at 0x0000 makes the session stop after just a few seconds, even before car starts taking amps.
	payload[46] = byte(maxCurrent)
	return &Datagram{Command: CmdChargeStart, Payload: payload[:]}
}

func (evse *Evse) StopCharge(params types.ChargeStopParams) (types.ChargeStopResult, error) {
	if !evse.IsOnline() {
		return types.ChargeStopResult{ErrorReason: types.ChargeStopErrorEvseOffline}, types.EvseOfflineError{Evse: evse}
	}
	if !evse.IsLoggedIn() {
		return types.ChargeStopResult{ErrorReason: types.ChargeStopErrorEvseNotLoggedIn}, types.EvseNotLoggedInError{Evse: evse}
	}

	var payload [47]byte
	if params.LineId == 0 {
		payload[0] = 1
	} else {
		payload[0] = params.LineId
	}
	WriteUserId(payload[1:17], params.UserId)

	stopChargeDatagram := &Datagram{
		Command: CmdChargeStop,
		Payload: payload[:],
	}
	sendErr := evse.SendDatagram(stopChargeDatagram)
	if sendErr != nil {
		return types.ChargeStopResult{
			ErrorReason:  types.ChargeStopErrorSendFailed,
			ErrorMessage: GetChargeStopErrorMessage(types.ChargeStopErrorSendFailed),
		}, sendErr
	}
	response, recvErr := evse.WaitForDatagram(5*time.Second, CmdChargeStartResponse)
	if recvErr != nil {
		return types.ChargeStopResult{
			ErrorReason:  types.ChargeStopErrorNoConfirmation,
			ErrorMessage: GetChargeStopErrorMessage(types.ChargeStopErrorNoConfirmation),
		}, recvErr
	}

	if CheckPayloadLength(response, evse, 5) {
		errorMessage := GetChargeStopErrorMessage(types.ChargeStopErrorNoConfirmation)
		return types.ChargeStopResult{
			ErrorReason:  types.ChargeStopErrorNoConfirmation,
			ErrorMessage: errorMessage,
		}, types.EvseInvalidDatagramError{Evse: evse, ResponseCommand: uint16(response.Command)}
	}

	errorReason := types.ChargeStopErrorReason(response.Payload[2])
	if errorReason != types.ChargeStopOK {
		errorMessage := GetChargeStopErrorMessage(errorReason)
		return types.ChargeStopResult{
			ErrorReason:  errorReason,
			ErrorMessage: errorMessage,
		}, types.EvseChargeStopError{Evse: evse, ErrorReason: errorReason, ErrorMessage: errorMessage}
	}

	return types.ChargeStopResult{
		ErrorReason: types.ChargeStopOK,
		LineId:      response.Payload[0],
	}, nil
}

func (evse *Evse) Login(password types.EmPassword) error {
	if password == "" {
		password = evse.password
		if password == "" {
			return types.EvseNoPasswordError{Evse: evse}
		}
	}

	requestLoginDatagram := &Datagram{
		Command:  CmdRequestLogin,
		Password: password,
		Payload:  []byte{0},
	}
	err := evse.SendDatagram(requestLoginDatagram)
	if err != nil {
		return err
	}

	response, err := evse.WaitForDatagram(5*time.Second, CmdLoginResponse, CmdPasswordErrorResponse)
	if err != nil {
		return err
	}
	if response.Command == CmdPasswordErrorResponse {
		return types.EvseInvalidPasswordError{Evse: evse}
	}

	evse.password = password

	loginConfirmDatagram := &Datagram{
		Command: CmdLoginConfirm,
		Payload: []byte{0},
	}
	err = evse.SendDatagram(loginConfirmDatagram)
	if err != nil {
		return err
	}

	wasLoggedIn := evse.IsLoggedIn()
	now := time.Now()
	evse.LastActiveLogin = &now
	if !wasLoggedIn {
		evse.QueueEvent(types.EvseLoggedIn)
	}

	// Fetch info, charge and config asynchronously after login.
	go func() { _ = evse.info.Fetch(0) }()
	go func() { _ = evse.charge.Fetch(0) }()
	go func() { _ = evse.config.Fetch(0) }()

	return nil
}

// AutoLogin attempts to log in to the EVSE if it is online, not logged in, and a password is set. It will
// keep trying to log in until successful or until the conditions are no longer met (e.g. EVSE goes offline or
// an explicit UsePassword / Login call succeeds in logging in).
func (evse *Evse) AutoLogin() {
	if evse.password == "" || !evse.IsOnline() || evse.IsLoggedIn() {
		return
	}
	err := evse.Login(evse.password)
	if err == nil {
		evse.communicator.Logger_.Debugf("[emproto4go] AutoLogin to EVSE %s successful", evse.Serial())
		return
	}
	evse.communicator.Logger_.Warnf("[emproto4go] AutoLogin to EVSE %s failed: %v (will retry)", evse.Serial(), err)
	go func() {
		time.Sleep(5 * time.Second)
		evse.AutoLogin()
	}()
}

func (evse *Evse) SendDatagram(datagram *Datagram) error {
	return evse.communicator.SendDatagram(evse, datagram)
}

// WaitForDatagram waits for a datagram with one of the specified commands to be received from the EVSE, and
// returns the first matching datagram received. If no datagram is received within the specified timeout, or the
// wait is interrupted (e.g. communicator is stopped), an error is returned.
func (evse *Evse) WaitForDatagram(timeout time.Duration, commands ...EmCommand) (*Datagram, error) {
	if len(commands) == 0 {
		return nil, nil
	}
	ch := make(chan *Datagram, 1)

	evse.waitersMutex.Lock()
	if evse.waiters == nil {
		evse.waiters = make(map[EmCommand][]chan *Datagram)
	}
	for _, cmd := range commands {
		evse.waiters[cmd] = append(evse.waiters[cmd], ch)
	}
	evse.waitersMutex.Unlock()

	// Listen for communicator stop
	stopped := make(chan struct{})
	go func() {
		for {
			if evse.communicator != nil && !evse.communicator.started {
				close(stopped)
				return
			}
			time.Sleep(50 * time.Millisecond)
		}
	}()

	select {
	case d := <-ch:
		return d, nil
	case <-time.After(timeout):
		// Cleanup waiter
		evse.waitersMutex.Lock()
		for _, cmd := range commands {
			chs := evse.waiters[cmd]
			for i, c := range chs {
				if c == ch {
					evse.waiters[cmd] = append(chs[:i], chs[i+1:]...)
					break
				}
			}
			if len(evse.waiters[cmd]) == 0 {
				delete(evse.waiters, cmd)
			}
		}
		evse.waitersMutex.Unlock()
		return nil, fmt.Errorf("timeout waiting for datagram")
	case <-stopped:
		// Cleanup waiter
		evse.waitersMutex.Lock()
		for _, cmd := range commands {
			chs := evse.waiters[cmd]
			for i, c := range chs {
				if c == ch {
					evse.waiters[cmd] = append(chs[:i], chs[i+1:]...)
					break
				}
			}
			if len(evse.waiters[cmd]) == 0 {
				delete(evse.waiters, cmd)
			}
		}
		evse.waitersMutex.Unlock()
		return nil, fmt.Errorf("communicator stopped while waiting for datagram")
	}
}

func (evse *Evse) Watch(eventTypes []types.EmEventType, channel chan<- types.EmEvent) types.EmEventWatcher {
	return evse.communicator.WatchImpl(evse, eventTypes, channel)
}

func (evse *Evse) QueueEvent(eventType types.EmEventType) {
	evse.communicator.QueueEvent(evse, eventType)
}

func (evse *Evse) DatagramReceived(datagram *Datagram, addr *net.Addr) {
	if datagram.Serial != evse.Serial() {
		return
	}
	evse.communicator.Logger_.Tracef("[emproto4go] <- RECV %+v from %v", datagram, *addr)

	wasOnline := evse.IsOnline()
	now := time.Now()
	evse.LastSeen = &now

	if !wasOnline {
		evse.QueueEvent(types.EvseOnline)
	}

	// Update addr (ip).
	if udpAddr, ok := (*addr).(*net.UDPAddr); ok {
		infoChanged := false
		if !evse.ip.Equal(udpAddr.IP) {
			evse.ip = udpAddr.IP
			infoChanged = true
		}
		if evse.port != udpAddr.Port {
			evse.port = udpAddr.Port
			infoChanged = true
		}
		if infoChanged {
			evse.QueueEvent(types.EvseInfoUpdated)
		}
	}

	// If we have a password, then start a login flow asynchronously.
	if !wasOnline && evse.password != "" {
		go evse.AutoLogin()
	}

	// Further processing of commands by handlers.
	if HandlerDelegator.Handle(evse, datagram) == 0 {
		evse.communicator.Logger_.Debugf("[emproto4go] No handler for command %s from EVSE %s", datagram.Command, evse.Serial())
	}

	// Unblock all waiters for this command
	evse.waitersMutex.Lock()
	if evse.waiters != nil {
		chs := evse.waiters[datagram.Command]
		for _, ch := range chs {
			select {
			case ch <- datagram:
			default:
			}
		}
		if len(chs) > 0 {
			delete(evse.waiters, datagram.Command)
		}
	}
	evse.waitersMutex.Unlock()
}

func (evse *Evse) Tick() {
	if !evse.IsLoggedIn() && evse.LastActiveLogin != nil {
		evse.LastActiveLogin = nil
		evse.QueueEvent(types.EvseLoggedOut)
	}

	if !evse.IsOnline() && evse.LastSeen != nil {
		evse.LastSeen = nil
		evse.QueueEvent(types.EvseOffline)
	}

	if evse.IsLoggedIn() {
		go func() { _ = evse.MutableCharge().Fetch(30 * time.Second) }()
		go func() { _ = evse.MutableInfo().Fetch(4 * time.Minute) }()
		go func() { _ = evse.MutableConfig().Fetch(3 * time.Minute) }()
	}
}

func (evse *Evse) String() string {
	return fmt.Sprintf("Evse{Label: %s, Serial: %s, MetaState: %v}",
		evse.Label(), evse.Serial(), evse.MetaState())
}
