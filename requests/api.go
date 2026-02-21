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

type CreateInputRequest struct {
	InputName string `json:"inputName"`
	Type      string `json:"type"`
	Address   string `json:"address,omitempty"`
	Login     string `json:"login,omitempty"`
	Password  string `json:"password,omitempty"`
	Modem     string `json:"modem,omitempty"`
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
