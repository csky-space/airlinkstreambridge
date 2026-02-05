package telemetry

import (
	"AirlinkStreamBridge/receivers"
	"time"

	"github.com/bluenviron/gomavlib/v3"
	"github.com/bluenviron/gomavlib/v3/pkg/dialects/ardupilotmega"
)

type UDPTelemetry struct {
	Enabled bool   `mapstructure:"enabled" json:"enabled" yaml:"enabled"`
	Address string `mapstructure:"address" json:"address" yaml:"address"`
	Port    int    `mapstructure:"port" json:"port" yaml:"port"`

	stopRequest *receivers.EventBroadcaster

	mavNode *gomavlib.Node
}

func NewDefaultUDPTelemetry(address string, port int) (UDPTelemetry, error) {
	telemetry := UDPTelemetry{
		mavNode: &gomavlib.Node{
			Endpoints: []gomavlib.EndpointConf{
				gomavlib.EndpointUDPClient{
					Address: address + ":" + string(rune(port)),
				},
			},
			Dialect:         ardupilotmega.Dialect,
			OutVersion:      gomavlib.V2,
			OutSystemID:     255,
			OutComponentID:  0,
			HeartbeatPeriod: 1 * time.Second,
		},
		stopRequest: receivers.NewEventBroadcaster(),
	}
	return telemetry, telemetry.mavNode.Initialize()
}

func (u *UDPTelemetry) Start() error {
	stopRequestSub := u.stopRequest.Subscribe()
	defer u.stopRequest.Unsubscribe(stopRequestSub)
	go func() {
		for evt := range u.mavNode.Events() {
			select {
			case <-stopRequestSub:
				return
			default:
			}
			if frm, ok := evt.(*gomavlib.EventFrame); ok {
				_ = frm // Process incoming frames if needed
			}
		}
	}()
	return nil
}

func (u *UDPTelemetry) Stop() error {
	u.mavNode.Close()
	return nil
}
