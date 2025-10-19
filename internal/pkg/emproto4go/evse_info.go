package emproto4go

import (
	"log"
	"time"

	"github.com/johnwoo-nl/emproto4go/internal/pkg/itypes"
	"github.com/johnwoo-nl/emproto4go/pkg/types"
)

type EvseInfo struct {
	evse        *Evse
	LastFetched *time.Time

	serial           types.EmSerial
	Brand_           string
	Model_           string
	HardwareVersion_ string
	SoftwareVersion_ string
	EvseType_        byte
	Phases_          types.EmPhases
	MaxPower_        types.Watts
	MaxCurrent_      types.Amps
	Feature_         uint32
	SupportNew_      uint32
	Byte70_          byte
}

func (info EvseInfo) Serial() types.EmSerial {
	return info.serial
}

func (info EvseInfo) EvseType() byte {
	return info.EvseType_
}

func (info EvseInfo) Brand() string {
	return info.Brand_
}

func (info EvseInfo) Model() string {
	return info.Model_
}
func (info EvseInfo) HardwareVersion() string {
	return info.HardwareVersion_
}

func (info EvseInfo) SoftwareVersion() string {
	return info.SoftwareVersion_
}

func (info EvseInfo) Phases() types.EmPhases {
	return info.Phases_
}

func (info EvseInfo) CanForceSinglePhase() bool {
	return info.EvseType_ >= 22 && info.EvseType_ <= 25 && info.Byte70_ < 11
}

func (info EvseInfo) MaxPower() types.Watts {
	return info.MaxPower_
}

func (info EvseInfo) MaxCurrent() types.Amps {
	return info.MaxCurrent_
}

func (info EvseInfo) Feature() uint32 {
	return info.Feature_
}

func (info EvseInfo) SupportNew() uint32 {
	return info.SupportNew_
}

func (info EvseInfo) Byte70() byte {
	return info.Byte70_
}

func (info EvseInfo) Fetch(maxAge time.Duration) error {
	if info.LastFetched != nil && time.Since(*info.LastFetched) < maxAge {
		return nil
	}
	if !info.evse.IsLoggedIn() {
		return types.EvseNotLoggedInError{Evse: info.evse}
	}

	datagram := Datagram{Command: itypes.CmdGetVersion, Payload: []byte{}}
	if sendErr := info.evse.SendDatagram(&datagram); sendErr != nil {
		return sendErr
	}
	_, recvErr := info.evse.WaitForDatagram(5*time.Second, itypes.CmdGetVersionResponse)
	if recvErr != nil && info.evse.communicator.Debug {
		log.Printf("[!] Failed to fetch info for EVSE %s: %v.", info.evse.Serial(), recvErr)
	}
	return recvErr
}
