package itypes

import (
	"fmt"

	"github.com/johnwoo-nl/emproto4go/pkg/types"
)

type EmCommand uint16

const (
	CmdLogin                 = EmCommand(0x0001)
	CmdLoginResponse         = EmCommand(0x0002)
	CmdLoginConfirm          = EmCommand(0x8001)
	CmdRequestLogin          = EmCommand(0x8002)
	CmdPasswordErrorResponse = EmCommand(0x0155)
	CmdHeading               = EmCommand(0x0003)
	CmdHeadingResponse       = EmCommand(0x8003)

	CmdSingleACStatus    = EmCommand(0x0004) // Sent by EVSE unsolicited periodically when logged in.
	CmdSingleACStatusAck = EmCommand(0x8004) // Ack for above.

	CmdSingleACChargingPublicAuto     = EmCommand(0x0005) // Sent by EVSE unsolicited periodically when logged in.
	CmdSingleACChargingAck            = EmCommand(0x8005) // Ack for above.
	CmdRequestSingleACCharging        = EmCommand(0x8006) // Sent by client to explicitly request status.
	CmdSingleACChargingStatusResponse = EmCommand(0x0006) // Response to above, same datagram layout as 0x0005.

	CmdGetVersion         = EmCommand(0x8106) // Request version info (some fields not present in CmdLogin).
	CmdGetVersionResponse = EmCommand(0x0106) // Version info.

	CmdSetAndGetLanguage                = EmCommand(0x810F)
	CmdSetAndGetLanguageResponse        = EmCommand(0x010F)
	CmdSetAndGetName                    = EmCommand(0x8108)
	CmdSetAndGetNameResponse            = EmCommand(0x0108)
	CmdSetAndGetOfflineCharge           = EmCommand(0x810D)
	CmdSetAndGetOfflineChargeResponse   = EmCommand(0x010D)
	CmdSetAndGetMaxCurrent              = EmCommand(0x8107)
	CmdSetAndGetMaxCurrentResponse      = EmCommand(0x0107)
	CmdSetAndGetTemperatureUnit         = EmCommand(0x8112)
	CmdSetAndGetTemperatureUnitResponse = EmCommand(0x0112)

	CmdChargeStart         = EmCommand(0x8007)
	CmdChargeStartResponse = EmCommand(0x0007)
	CmdChargeStop          = EmCommand(0x8008)
	CmdChargeStopResponse  = EmCommand(0x0008)
)

var emCommandNames = map[EmCommand]string{
	CmdLogin:                            "CmdLogin",
	CmdLoginResponse:                    "CmdLoginResponse",
	CmdLoginConfirm:                     "CmdLoginConfirm",
	CmdRequestLogin:                     "CmdRequestLogin",
	CmdPasswordErrorResponse:            "CmdPasswordErrorResponse",
	CmdHeading:                          "CmdHeading",
	CmdHeadingResponse:                  "CmdHeadingResponse",
	CmdSingleACStatus:                   "CmdSingleACStatus",
	CmdSingleACStatusAck:                "CmdSingleACStatusAck",
	CmdSingleACChargingPublicAuto:       "CmdSingleACChargingPublicAuto",
	CmdSingleACChargingAck:              "CmdSingleACChargingAck",
	CmdGetVersion:                       "CmdGetVersion",
	CmdGetVersionResponse:               "CmdGetVersionResponse",
	CmdSetAndGetLanguage:                "CmdSetAndGetLanguage",
	CmdSetAndGetLanguageResponse:        "CmdSetAndGetLanguageResponse",
	CmdSetAndGetName:                    "CmdSetAndGetName",
	CmdSetAndGetNameResponse:            "CmdSetAndGetNameResponse",
	CmdSetAndGetOfflineCharge:           "CmdSetAndGetOfflineCharge",
	CmdSetAndGetOfflineChargeResponse:   "CmdSetAndGetOfflineChargeResponse",
	CmdSetAndGetMaxCurrent:              "CmdSetAndGetMaxCurrent",
	CmdSetAndGetMaxCurrentResponse:      "CmdSetAndGetMaxCurrentResponse",
	CmdSetAndGetTemperatureUnit:         "CmdSetAndGetTemperatureUnit",
	CmdSetAndGetTemperatureUnitResponse: "CmdSetAndGetTemperatureUnitResponse",
	CmdRequestSingleACCharging:          "CmdRequestSingleACCharging",
	CmdSingleACChargingStatusResponse:   "CmdSingleACChargingStatusResponse",
	CmdChargeStart:                      "CmdChargeStart",
	CmdChargeStartResponse:              "CmdChargeStartResponse",
	CmdChargeStop:                       "CmdChargeStop",
	CmdChargeStopResponse:               "CmdChargeStopResponse",
}

func (e EmCommand) String() string {
	str := fmt.Sprintf("0x%04x", uint16(e))
	if name, ok := emCommandNames[e]; ok {
		return str + ":" + name
	}
	return str
}

var EmChargeStartErrorMessages = map[types.ChargeStartErrorReason]string{
	types.ChargeStartOK:                                "No error",
	types.ChargeStartErrorPlugNotProperlyInserted:      "The charging plug is not plugged in properly",
	types.ChargeStartErrorSystemError:                  "System error",
	types.ChargeStartErrorAlreadyCharging:              "Already currently charging",
	types.ChargeStartErrorSystemMaintenance:            "System maintenance",
	types.ChargeStartErrorIncorrectSetFee:              "Incorrect set fee",
	types.ChargeStartErrorIncorrectSetPowerConsumption: "Incorrect set power consumption",
	types.ChargeStartErrorIncorrectSetTime:             "Incorrect set time",
	types.ChargeStartErrorAlreadyReserved:              "Charging reservation already active",

	types.ChargeStartErrorEvseOffline:     "EVSE is offline",
	types.ChargeStartErrorEvseNotLoggedIn: "EVSE is not logged in",
	types.ChargeStartErrorSendFailed:      "Failed to send start command to EVSE",
	types.ChargeStartErrorNoConfirmation:  "No confirmation received from EVSE (charge could still have started)",
	types.ChargeStartErrorUnknown:         "Unknown reason",
}

var EmChargeStopErrorMessages = map[types.ChargeStopErrorReason]string{
	types.ChargeStopOK: "No error",

	types.ChargeStopErrorEvseOffline:     "EVSE is offline",
	types.ChargeStopErrorEvseNotLoggedIn: "EVSE is not logged in",
	types.ChargeStopErrorSendFailed:      "Failed to send stop command to EVSE",
	types.ChargeStopErrorNoConfirmation:  "No confirmation received from EVSE (charge could still have stopped)",
	types.ChargeStopErrorUnknown:         "Unknown reason",
}
