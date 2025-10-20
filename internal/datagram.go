package internal

import (
	"encoding/binary"
	"fmt"

	"github.com/johnwoo-nl/emproto4go/types"
)

type Datagram struct {
	Key      byte
	Serial   types.EmSerial
	Password types.EmPassword
	Command  EmCommand
	Payload  []byte
}

func (datagram *Datagram) Encode() ([]byte, error) {
	if datagram == nil {
		return nil, fmt.Errorf("datagram is nil")
	}
	packetLen := 25 + len(datagram.Payload)
	data := make([]byte, packetLen)
	binary.BigEndian.PutUint16(data[0:2], 0x0601)
	binary.BigEndian.PutUint16(data[2:4], uint16(packetLen))
	data[4] = datagram.Key

	// Serial: convert 16 hex chars to 8 bytes
	serialStr := string(datagram.Serial[:])
	if len(serialStr) != 16 {
		return nil, fmt.Errorf("serial must be 16 hex characters, got %d", len(serialStr))
	}
	for i := 0; i < 8; i++ {
		var b byte
		_, err := fmt.Sscanf(serialStr[i*2:i*2+2], "%02X", &b)
		if err != nil {
			_, errLower := fmt.Sscanf(serialStr[i*2:i*2+2], "%02x", &b)
			if errLower != nil {
				return nil, fmt.Errorf("invalid serial hex at position %d: %v", i, err)
			}
		}
		data[5+i] = b
	}

	// Password: if not 6 bytes, set to null bytes
	if len(datagram.Password) != 6 {
		for i := 13; i < 19; i++ {
			data[i] = 0x00
		}
	} else {
		copy(data[13:19], datagram.Password[:])
	}

	binary.BigEndian.PutUint16(data[19:21], uint16(datagram.Command))
	copy(data[21:21+len(datagram.Payload)], datagram.Payload)

	var checksum uint16
	for _, b := range data[:len(data)-4] {
		checksum = (checksum + uint16(b)) % 0xffff
	}
	binary.BigEndian.PutUint16(data[len(data)-4:len(data)-2], checksum)
	binary.BigEndian.PutUint16(data[len(data)-2:], 0x0F02)
	return data, nil
}

func Decode(data []byte) (*Datagram, error) {
	// This is not an EvseMaster datagram. Don't return an error, just nil to indicate not handled.
	if len(data) < 25 {
		return nil, nil
	}

	// This is not an EvseMaster datagram. Don't return an error, just nil to indicate not handled.
	magicHeader := binary.BigEndian.Uint16(data[0:2])
	if magicHeader != 0x0601 {
		return nil, nil
	}

	packetLen := binary.BigEndian.Uint16(data[2:4])
	if packetLen != uint16(len(data)) {
		return nil, nil
	}

	packetChecksum := binary.BigEndian.Uint16(data[len(data)-4 : len(data)-2])
	var computedChecksum uint16
	for _, b := range data[:len(data)-4] {
		computedChecksum = (computedChecksum + uint16(b)) % 0xffff
	}
	if computedChecksum != packetChecksum {
		return nil, InvalidDatagramError{Message: fmt.Sprintf("checksum mismatch, computed %04x does not match %04x from packet", computedChecksum, packetChecksum)}
	}

	datagram := Datagram{
		Key:      data[4],
		Serial:   types.EmSerial(fmt.Sprintf("%02x", data[5:13])),
		Password: types.EmPassword(ReadString(data[13:19])),
		Command:  EmCommand(binary.BigEndian.Uint16(data[19:21])),
		Payload:  data[21 : len(data)-4],
	}

	return &datagram, nil
}

func (datagram *Datagram) String() string {
	pwd := "(not set)"
	if len(datagram.Password) > 0 {
		pwd = "(set)"
	}

	// Format Payload as space-separated hex bytes
	payloadStr := ""
	for i, b := range datagram.Payload {
		if i > 0 {
			payloadStr += " "
		}
		payloadStr += fmt.Sprintf("%02x", b)
	}
	if payloadStr == "" {
		payloadStr = "(empty)"
	}

	return fmt.Sprintf(
		"Datagram{Command:%s Serial:%s Key:%d Password:%s Payload:%s PayloadLen=%d}",
		datagram.Command, datagram.Serial, datagram.Key, pwd, payloadStr, len(datagram.Payload),
	)
}
