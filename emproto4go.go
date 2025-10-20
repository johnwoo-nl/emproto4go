package emproto4go

import (
	"github.com/johnwoo-nl/emproto4go/internal"
	_ "github.com/johnwoo-nl/emproto4go/internal/handlers" // for side effects (handlers self-register via init)
	"github.com/johnwoo-nl/emproto4go/types"
)

// CreateCommunicator creates a new communicator instance.
// This is the entry point of the library.
//
//lint:ignore U1000 exported API
//goland:noinspection GoUnusedExportedFunction
func CreateCommunicator(appName types.UserId) types.EmCommunicator {
	return internal.CreateCommunicator(appName)
}
