package internal

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/johnwoo-nl/emproto4go/types"
)

var locShanghai, _ = time.LoadLocation("Asia/Shanghai")

func CompareAndSet[T comparable](old *T, new T) bool {
	if *old != new {
		*old = new
		return true
	}
	return false
}

// SliceEqual compares two slices for equality, ignoring order or duplicity of elements.
func SliceEqual[T comparable](a, b []T) bool {
	if len(a) != len(b) {
		return false
	}
	count := make(map[T]int)
	for _, err := range a {
		count[err]++
	}
	for _, err := range b {
		if count[err] == 0 {
			return false
		}
		count[err]--
	}
	return true
}

func ReadString(data []byte) string {
	return string(bytes.Trim(data, "\x00\xFF\x20"))
}

func ReadDurationMinutes(data []byte, offset int) *time.Duration {
	value := binary.BigEndian.Uint16(data[offset : offset+2])
	if value == 0xFFFF {
		return nil
	}
	duration := time.Duration(value) * time.Minute
	return &duration
}

func ReadDurationSeconds(data []byte, offset int) time.Duration {
	value := binary.BigEndian.Uint32(data[offset : offset+4])
	return time.Duration(value) * time.Second
}

func ReadEnergy32(data []byte, offset int) *types.KWh {
	value := binary.BigEndian.Uint32(data[offset : offset+4])
	if value == 0xFFFFFFFF {
		return nil
	}
	energy := types.KWh(float64(value) * 0.01)
	return &energy
}

func ReadEnergy16(data []byte, offset int) *types.KWh {
	value := binary.BigEndian.Uint16(data[offset : offset+2])
	if value == 0xFFFF {
		return nil
	}
	energy := types.KWh(float64(value) * 0.01)
	return &energy
}

func ReadTemperature(data []byte, offset int) types.TempCelsius {
	temp := binary.BigEndian.Uint16(data[offset : offset+2])
	if temp == 0xFFFF {
		return -1.0
	}
	return types.TempCelsius(float32(temp-20000) * 0.01)
}

func ReadTimestamp(data []byte, offset int) *time.Time {
	timestamp := binary.BigEndian.Uint32(data[offset : offset+4])
	return EmTimestampToTime(timestamp)
}

func TimeToEmTimestamp(time *time.Time) uint32 {
	if time == nil || time.UnixMilli() == 0 {
		return 0xFFFFFFFF
	}

	offset := getOffsetBetweenLocalAndShanghai(*time)
	return uint32(time.Unix() - int64(offset))
}

func EmTimestampToTime(timestamp uint32) *time.Time {
	if timestamp == 0 || timestamp == 0xFFFFFFFF {
		return nil
	}

	shanghaiTime := time.Unix(int64(timestamp), 0).In(locShanghai)
	offset := getOffsetBetweenLocalAndShanghai(shanghaiTime)
	adjusted := shanghaiTime.UnixMilli() + int64(offset*1000)
	timeInLocal := time.UnixMilli(adjusted).In(time.Local)
	return &timeInLocal
}

// Returns the offset in seconds between local timezone and Asia/Shanghai at the given time.
func getOffsetBetweenLocalAndShanghai(t time.Time) int {
	_, localOffset := t.In(time.Local).Zone()
	_, shanghaiOffset := t.In(locShanghai).Zone()
	return shanghaiOffset - localOffset
}

func CheckPayloadLength(datagram *Datagram, evse *Evse, minLength int) bool {
	if len(datagram.Payload) < minLength {
		evse.communicator.Logger_.Warnf("[emproto4go] Received invalid datagram (command 0x%02X) from EVSE %+v: payload too short (need at least %d bytes, got %d)",
			datagram.Command, evse, minLength, len(datagram.Payload))
		return true
	}
	return false
}

func WriteUserId(buffer []byte, userId types.UserId) {
	if len(userId) > 16 {
		userId = userId[:16]
	}
	copy(buffer, userId)
}

func MakeChargeId(chargeIdSuffix string) types.ChargeId {
	// The OEM app generates a charge ID in the format yyyyMMddHHmm (using current time, in Asia/Shanghai timezone)
	// with 4 random characters appended. The OEM app does not actually use the time part besides showing it, but it
	// does do something with the month part (some January/February correction when displaying charge sessions). So
	// we will use the yyyyMMdd prefix, but omit the time part, so we have 8 characters for the app-provided chargeId
	// suffix. Apps can then correlate the last 8 characters with their stored charge ID, while we still maintain
	// compatibility with the OEM app.
	now := time.Now().In(locShanghai)
	prefix := now.Format("20060102")
	// If no app-provided suffix is given, add the time part anyway and suffix a random 4-digit suffix, like the OEM app does.
	if len(chargeIdSuffix) == 0 {
		prefix = now.Format("200601021504")
		b := make([]byte, 4)
		for i := range b {
			b[i] = '0' + byte(time.Now().UnixNano()%10)
		}
		chargeIdSuffix = string(b)
	} else if len(chargeIdSuffix) > 8 {
		chargeIdSuffix = chargeIdSuffix[:8]
	}
	chargeId := types.ChargeId(prefix + chargeIdSuffix)
	return chargeId
}

func GetChargeStartErrorReasonMessage(errorReason types.ChargeStartErrorReason) string {
	if msg, ok := EmChargeStartErrorMessages[errorReason]; ok {
		return msg
	}
	return fmt.Sprintf("%s (%d)", EmChargeStartErrorMessages[types.ChargeStartErrorUnknown], errorReason)
}

func GetChargeStopErrorMessage(errorReason types.ChargeStopErrorReason) string {
	if msg, ok := EmChargeStopErrorMessages[errorReason]; ok {
		return msg
	}
	return fmt.Sprintf("%s (%d)", EmChargeStopErrorMessages[types.ChargeStopErrorUnknown], errorReason)
}
