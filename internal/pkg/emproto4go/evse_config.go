package emproto4go

import (
	"log"
	"time"

	"github.com/johnwoo-nl/emproto4go/internal/pkg/itypes"
	"github.com/johnwoo-nl/emproto4go/pkg/types"
)

type EvseConfig struct {
	evse        *Evse
	LastFetched *time.Time

	Name_            string
	Language_        types.EmLanguage
	TemperatureUnit_ types.EmTemperatureUnit
	OfflineCharge_   bool
	MaxCurrent_      types.Amps
}

type result struct {
	name string
	err  error
}

func (config *EvseConfig) Name() string {
	return config.Name_
}

func (config *EvseConfig) Language() types.EmLanguage {
	return config.Language_
}

func (config *EvseConfig) TemperatureUnit() types.EmTemperatureUnit {
	return config.TemperatureUnit_
}

func (config *EvseConfig) CanOfflineCharge() bool {
	return config.OfflineCharge_
}

func (config *EvseConfig) MaxCurrent() types.Amps {
	return config.MaxCurrent_
}

func (config *EvseConfig) Fetch(maxAge time.Duration) error {
	if config.LastFetched != nil && time.Since(*config.LastFetched) < maxAge {
		return nil
	}
	if !config.evse.IsLoggedIn() {
		return types.EvseNotLoggedInError{Evse: config.evse}
	}

	resultsChan := make(chan result, 5)

	// Send out the GET requests and wait for responses in parallel.
	// Note that we don't actually process the incoming config values here; they are handled by ConfigHandler.
	go func() { resultsChan <- config.get("Name", itypes.CmdSetAndGetName, 32) }()
	go func() { resultsChan <- config.get("Language", itypes.CmdSetAndGetLanguage, 1) }()
	go func() { resultsChan <- config.get("TemperatureUnit", itypes.CmdSetAndGetTemperatureUnit, 1) }()
	go func() { resultsChan <- config.get("OfflineCharge", itypes.CmdSetAndGetOfflineCharge, 1) }()
	go func() { resultsChan <- config.get("MaxCurrent", itypes.CmdSetAndGetMaxCurrent, 1) }()

	// Wait for all fetches to complete (or timeout), and collect errors with field names.
	var failed []string
	for i := 0; i < 5; i++ {
		result := <-resultsChan
		if result.err != nil {
			failed = append(failed, result.name+": "+result.err.Error())
			if config.evse.communicator.Debug {
				log.Printf("[!] Failed getting config item %s for EVSE %+v: %v", result.name, config.evse, result.err)
			}
		}
	}

	if len(failed) == 0 {
		now := time.Now()
		config.LastFetched = &now
		return nil
	}

	return types.EvseConfigFetchError{Evse: config.evse, FailedFields: failed}
}

func (config *EvseConfig) SetName(name string) error {
	// Ensure name contains only ASCII characters and is at most 11 bytes
	asciiName := make([]byte, 0, 11)
	for i := 0; i < len(name) && len(asciiName) < 11; i++ {
		if name[i] <= 0x7F {
			asciiName = append(asciiName, name[i])
		}
	}
	value := make([]byte, 32)
	copy(value[:4], []byte{'A', 'C', 'P', '#'})
	copy(value[4:], asciiName)

	if err := config.set(itypes.CmdSetAndGetName, value); err != nil {
		return err
	}
	config.Name_ = string(asciiName)
	config.evse.QueueEvent(types.EvseConfigUpdated)
	return nil
}

func (config *EvseConfig) SetLanguage(language types.EmLanguage) error {
	if err := config.set(itypes.CmdSetAndGetLanguage, []byte{byte(language)}); err != nil {
		return err
	}
	config.Language_ = language
	config.evse.QueueEvent(types.EvseConfigUpdated)
	return nil
}

func (config *EvseConfig) SetTemperatureUnit(unit types.EmTemperatureUnit) error {
	if err := config.set(itypes.CmdSetAndGetTemperatureUnit, []byte{byte(unit)}); err != nil {
		return err
	}
	config.TemperatureUnit_ = unit
	config.evse.QueueEvent(types.EvseConfigUpdated)
	return nil
}

func (config *EvseConfig) SetOfflineCharge(offlineCharge bool) error {
	offlineChargeByte := byte(1)
	if offlineCharge {
		offlineChargeByte = byte(0)
	}
	if err := config.set(itypes.CmdSetAndGetOfflineCharge, []byte{offlineChargeByte}); err != nil {
		return err
	}
	config.OfflineCharge_ = offlineCharge
	config.evse.QueueEvent(types.EvseConfigUpdated)
	return nil
}

func (config *EvseConfig) SetMaxCurrent(maxCurrent types.Amps) error {
	if err := config.set(itypes.CmdSetAndGetMaxCurrent, []byte{byte(maxCurrent)}); err != nil {
		return err
	}
	config.MaxCurrent_ = maxCurrent
	config.evse.QueueEvent(types.EvseConfigUpdated)
	return nil
}

func (config *EvseConfig) get(name string, command itypes.EmCommand, valueLen uint) result {
	if !config.evse.IsLoggedIn() {
		return result{name: name, err: types.EvseNotLoggedInError{Evse: config.evse}}
	}
	payload := make([]byte, 1+valueLen)
	payload[0] = 0x02 // GET the config item; all share the same GET=2/SET=1 value.
	datagram := Datagram{Command: command, Payload: payload}
	if sendErr := config.evse.SendDatagram(&datagram); sendErr != nil {
		return result{name: name, err: sendErr}
	}
	responseCommand := command - 0x8000
	_, recvErr := config.evse.WaitForDatagram(8*time.Second, responseCommand)
	return result{name: name, err: recvErr}
}

func (config *EvseConfig) set(command itypes.EmCommand, value []byte) error {
	if !config.evse.IsLoggedIn() {
		return types.EvseNotLoggedInError{Evse: config.evse}
	}
	payload := make([]byte, 1+len(value))
	payload[0] = 0x01 // SET the config item; all share the same GET=2/SET=1 value.
	copy(payload[1:], value)
	datagram := Datagram{Command: command, Payload: payload}
	if sendErr := config.evse.SendDatagram(&datagram); sendErr != nil {
		return sendErr
	}
	responseCommand := command - 0x8000
	_, recvErr := config.evse.WaitForDatagram(5*time.Second, responseCommand)
	return recvErr
}
