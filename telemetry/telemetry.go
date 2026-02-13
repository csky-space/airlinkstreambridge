package telemetry

type Telemetry struct {
	onData func(data []byte) error
}

func NewTelemetry(onData func(data []byte) error) (*Telemetry, error) {
	return &Telemetry{onData: onData}, nil
}
