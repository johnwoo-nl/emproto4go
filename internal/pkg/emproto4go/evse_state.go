package emproto4go

import (
	"github.com/johnwoo-nl/emproto4go/pkg/types"
)

type EvseState struct {
	LineId_            types.LineId
	CurrentPower_      types.Watts
	EnergyCounter_     types.KWh
	L1Voltage_         types.Volts
	L1Current_         types.Amps
	L2Voltage_         types.Volts
	L2Current_         types.Amps
	L3Voltage_         types.Volts
	L3Current_         types.Amps
	InnerTemp_         types.TempCelsius
	OuterTemp_         types.TempCelsius
	EmergencyBtnState_ types.EmEmergencyBtnState
	CurrentState_      types.EmCurrentState
	GunState_          types.EmGunState
	OutputState_       types.EmOutputState
	Errors_            []types.EmError
	NewProtocol_       bool
}

func (evse EvseState) LineId() types.LineId {
	return evse.LineId_
}

func (evse EvseState) CurrentPower() types.Watts {
	return evse.CurrentPower_
}

func (evse EvseState) EnergyCounter() types.KWh {
	return evse.EnergyCounter_
}

func (evse EvseState) L1Voltage() types.Volts {
	return evse.L1Voltage_
}

func (evse EvseState) L1Current() types.Amps {
	return evse.L1Current_
}

func (evse EvseState) L2Voltage() types.Volts {
	return evse.L2Voltage_
}

func (evse EvseState) L2Currrent() types.Amps {
	return evse.L2Current_
}

func (evse EvseState) L3Voltage() types.Volts {
	return evse.L3Voltage_
}

func (evse EvseState) L3Current() types.Amps {
	return evse.L3Current_
}

func (evse EvseState) InnerTemp() types.TempCelsius {
	return evse.InnerTemp_
}

func (evse EvseState) OuterTemp() types.TempCelsius {
	return evse.OuterTemp_
}

func (evse EvseState) EmergencyBtnState() types.EmEmergencyBtnState {
	return evse.EmergencyBtnState_
}

func (evse EvseState) CurrentState() types.EmCurrentState {
	return evse.CurrentState_
}

func (evse EvseState) GunState() types.EmGunState {
	return evse.GunState_
}

func (evse EvseState) OutputState() types.EmOutputState {
	return evse.OutputState_
}

func (evse EvseState) Errors() []types.EmError {
	return evse.Errors_
}

func (evse EvseState) IsNewProtocol() bool {
	return evse.NewProtocol_
}
