package emproto4go

import (
	"github.com/johnwoo-nl/emproto4go/internal/pkg/emproto4go"
	_ "github.com/johnwoo-nl/emproto4go/internal/pkg/emproto4go/handlers" // for side effects
	"github.com/johnwoo-nl/emproto4go/pkg/types"
)

//lint:ignore U1000 exported API
//goland:noinspection GoUnusedExportedFunction
func CreateCommunicator(appName types.UserId, debug bool) types.EmCommunicator {
	return emproto4go.CreateCommunicator(appName, debug)
}
