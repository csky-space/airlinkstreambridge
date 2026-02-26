package requests

type DefaultReceiverRequest struct {
	HostName      string `json:"hostName"`
	ModemName     string `json:"modemName"`
	Password      string `json:"password"`
	UDPPort       int    `json:"UDPPort"`
	IcePolicyType string `json:"IcePolicy"`
}

type AssignRequest struct {
	InputName  string `json:"inputName"`
	OutputName string `json:"outputName"`
}

type DismissRequest struct {
	InputName  string `json:"inputName"`
	OutputName string `json:"outputName"`
}

type CreateOutputRequest struct {
	OutputName string `json:"outputName"`
	Type       string `json:"type"`
	Address    string `json:"address,omitempty"`
}

type createInputBase struct {
	InputName string `json:"inputName"`
	Type      string `json:"type"`
}

type createInputNet struct {
	Address string `json:"address,omitempty"`
}

type createInputAstra struct {
	Login    string `json:"login,omitempty"`
	Password string `json:"password,omitempty"`
	Modem    string `json:"modem,omitempty"`
}

type createInputAstraWebrtc struct {
	ConfigureFromModem string `json:"configureFromModem,omitempty"`
}

type CreateInputRequest struct {
	createInputBase
	createInputNet
	createInputAstra
	createInputAstraWebrtc
}

type RemoveOutputRequest struct {
	OutputName string `json:"outputName"`
}

type RemoveInputRequest struct {
	InputName string `json:"inputName"`
}

type StartOutputRequest struct {
	OutputName string `json:"outputName"`
}

type StartInputRequest struct {
	InputName string `json:"inputName"`
}

type TransferIODeviceRequest struct {
	Input  string `json:"inputName"`
	Output string `json:"outputName"`
}

type ioConfBase struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type ioNet struct {
	Address    string   `json:"address,omitempty"`
	TransferTo []string `json:"transferTo,omitempty"`
}

type ioProxy struct {
	Assign   []string `json:"assign,omitempty"`
	Activate bool     `json:"activate,omitempty"`
}

type ioAstra struct {
	Login          string `json:"login,omitempty"`
	Password       string `json:"password,omitempty"`
	AstraModemType string `json:"astraModemType,omitempty"`
}

type ioWebrtc struct {
	ConfigureFromModem string `json:"configureFromModem,omitempty"`
	IceTransportPolicy string `json:"iceTransportPolicy,omitempty"`
}

type IOConf struct {
	ioConfBase
	ioNet
	ioProxy
	ioAstra
	ioWebrtc
}

type SingleRequest struct {
	IOs []IOConf
}
