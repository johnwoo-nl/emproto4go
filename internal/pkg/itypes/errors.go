package itypes

import (
	"fmt"

	"github.com/johnwoo-nl/emproto4go/pkg/types"
)

type InvalidDatagramError struct {
	Message string
}

func (err InvalidDatagramError) Error() string {
	return fmt.Sprintf("Invalid datagram: %s", err.Message)
}

type DatagramSendError struct {
	Evse    types.EmEvse
	Command EmCommand
	Message string
}

func (err DatagramSendError) Error() string {
	return fmt.Sprintf("Failed to send datagram with command %v to %s: %s", err.Command, err.Evse.Serial(), err.Message)
}
