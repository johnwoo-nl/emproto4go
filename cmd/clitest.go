package main

import (
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/johnwoo-nl/emproto4go"
	"github.com/johnwoo-nl/emproto4go/types"
)

func main() {
	if len(os.Args) > 1 && (strings.ToLower(os.Args[1]) == "help" || strings.ToLower(os.Args[1]) == "--help" || strings.ToLower(os.Args[1]) == "-h") {
		log.Printf("Usage: %s [serial=password] [info] [start[=amps] | stop] [debug]", filepath.Base(os.Args[0]))
		log.Printf("  serial:   EVSE serial number (optional, prints only basic info otherwise)")
		log.Printf("  password: EVSE password (optional, prints only basic info otherwise)")
		log.Printf("  compat:   Print some EVSE compatibility info useful for debugging")
		log.Printf("  start:    Start charging after login")
		log.Printf("  amps:     Maximum current in amps (default: 6A)")
		log.Printf("  stop:     Stop charging after login")
		log.Printf("  debug:    Enable debug logging (includes sent/received datagrams)")
		return
	}

	debug := false
	compat := false
	serial := types.EmSerial("")
	password := types.EmPassword("")
	start := false
	amps := types.Amps(6)
	stop := false

	for _, arg := range os.Args[1:] {
		if strings.Contains(arg, "=") {
			parts := strings.SplitN(arg, "=", 2)
			if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
				if parts[0] == "start" {
					start = true
					ampsParsed, err := strconv.Atoi(parts[1])
					if err != nil {
						panic(err)
					}
					amps = types.Amps(ampsParsed)
				} else {
					serial = types.EmSerial(parts[0])
					password = types.EmPassword(parts[1])
				}
			}
		} else if arg == "start" {
			start = true
		} else if arg == "stop" {
			stop = true
		} else if arg == "debug" {
			debug = true
		} else if arg == "compat" || arg == "compatibility" {
			compat = true
		}
	}

	communicator := emproto4go.CreateCommunicator("emproto4go_test", debug)
	err := communicator.Start()
	if err != nil {
		log.Printf("Cannot start communicator: %v", err)
		return
	}
	defer communicator.Stop()

	if serial != "" && password != "" {
		_ = communicator.DefineEvse(serial).UsePassword(password)
	}

	// Create a channel to receive events
	ch := make(chan types.EmEvent, 10)
	// Watch all event types (empty slice means all types)
	watcher := communicator.Watch(nil, []types.EmEventType{}, ch)
	defer watcher.Stop()

	c := make(chan os.Signal, 1)

	// Goroutine to log received events
	go func() {
		for event := range ch {
			if event.Type == types.EvseInfoUpdated {
				log.Printf("[%v] Evse=%+v, Info=%+v", event.Type, event.Evse, event.Evse.Info())
			} else if event.Type == types.EvseStateUpdated {
				log.Printf("[%v] Evse=%+v, State=%+v", event.Type, event.Evse, event.Evse.State())
			} else if event.Type == types.EvseChargeUpdated {
				log.Printf("[%v] Evse=%+v, Charge=%+v", event.Type, event.Evse, event.Evse.Charge())
			} else if event.Type == types.EvseConfigUpdated {
				log.Printf("[%v] Evse=%+v, Config=%+v", event.Type, event.Evse, event.Evse.Config())
			} else {
				log.Printf("[%v] Evse=%+v", event.Type, event.Evse)
			}

			if event.Type == types.EvseLoggedIn {
				if start {
					go func() {
						time.Sleep(5 * time.Second)
						result, err := event.Evse.StartCharge(types.ChargeStartParams{
							MaxCurrent: amps,
						})
						if err != nil {
							log.Printf("Error starting charge: %v", err)
						} else {
							log.Printf("Charge started successfully; result: %+v", result)
						}
					}()
				} else if stop {
					go func() {
						time.Sleep(5 * time.Second)
						result, err := event.Evse.StopCharge(types.ChargeStopParams{})
						if err != nil {
							log.Printf("Error stopping charge: %v", err)
						} else {
							log.Printf("Charge stopped successfully; result: %+v", result)
						}
					}()
				}

				if compat {
					log.Printf("Working on it...")
					// Wait a bit for the version datagrams, then print compatibility info.
					go func() {
						time.Sleep(5 * time.Second)
						printCompatInfo(event.Evse)
						c <- os.Interrupt
					}()
				}
			}
		}
	}()

	// Wait for Ctrl+C (SIGINT) or SIGTERM
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	log.Println("Press Ctrl+C to exit.")
	<-c
	log.Println("Stopping...")
}

func printCompatInfo(evse types.EmEvse) {
	info := evse.Info()
	config := evse.Config()
	charge := evse.Charge()

	log.Printf("╔════════════════════════════════════════╗")
	log.Printf("║     EVSE Compatibility Information     ║")
	log.Printf("╠════════════════════════════════════════╣")
	log.Printf("║ Brand:                %16s ║", info.Brand())
	log.Printf("║ Model:                %16s ║", info.Model())
	log.Printf("║ EVSE Type:            %16d ║", info.EvseType())
	log.Printf("║ Serial Number:        %16s ║", info.Serial())
	log.Printf("║ Meta State:           %16s ║", evse.MetaState())
	log.Printf("║ Hardware Version:     %16s ║", info.HardwareVersion())
	log.Printf("║ Software Version:     %16s ║", info.SoftwareVersion())
	log.Printf("║ Phases:               %16d ║", info.Phases())
	log.Printf("║ Can Force Single Phase: %14t ║", info.CanForceSinglePhase())
	log.Printf("║ Max Power:            %14d W ║", info.MaxPower())
	log.Printf("║ Max Current:          %14d A ║", int(info.MaxCurrent()))
	log.Printf("║ Supported Features:         0x%08X ║", info.Feature())
	log.Printf("║ Supported New:              0x%08X ║", info.SupportNew())
	log.Printf("║ Byte70:                           0x%02X ║", info.Byte70())
	log.Printf("╠═════════════════CONFIG═════════════════╣")
	log.Printf("║ Configured Name:      %16s ║", config.Name())
	log.Printf("║ Language:             %16d ║", config.Language())
	log.Printf("║ Temperature Unit:     %16d ║", config.TemperatureUnit())
	log.Printf("║ Configured Max Current: %12d A ║", int(config.MaxCurrent()))
	log.Printf("╠═════════════════CHARGE═════════════════╣")
	log.Printf("║ ID:                   %16s ║", string(charge.ChargeId()))
	log.Printf("║ State:                %16d ║", charge.ChargeState())
	log.Printf("║ Started By:           %16s ║", string(charge.UserId()))
	log.Printf("║ Charge Max Current:   %14d A ║", int(charge.MaxCurrent()))
	log.Printf("║ Port:                 %16d ║", charge.Port())
	log.Printf("║ Charge Type:          %16d ║", charge.ChargeType())
	log.Printf("║ Duration:             %16v ║", charge.Duration())
	log.Printf("║ Charged Energy:     %14.2f kWh ║", float32(charge.ChargedEnergy()))
	if charge.MaxDuration() == nil {
		log.Printf("║ Max Duration:         %16s ║", "not limited")
	} else {
		log.Printf("║ Max Duration:         %16v ║", *charge.MaxDuration())
	}
	if charge.MaxEnergy() == nil {
		log.Printf("║ Max Energy:           %16s ║", "not limited")
	} else {
		log.Printf("║ Max Energy:       %14.2f kWh ║", float32(*charge.MaxEnergy()))
	}
	log.Printf("║ Start Time:                            ║")
	log.Printf("║ %38v ║", charge.StartTime())
	log.Printf("║ Reservation Time:                      ║")
	log.Printf("║ %38v ║", charge.ReservationTime())
	log.Printf("╚════════════════════════════════════════╝")
}
