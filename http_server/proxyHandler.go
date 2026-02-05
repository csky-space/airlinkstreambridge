package httpserver

import (
	"AirlinkStreamBridge/proxy"
	"AirlinkStreamBridge/registry"
	"AirlinkStreamBridge/requests"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
)

type ProxyHandler struct {
	proxies  map[string]proxy.Proxy
	_proxy   proxy.Proxy
	registry *registry.Registry
}

func NewProxyHandler() *ProxyHandler {
	return &ProxyHandler{_proxy: *proxy.NewProxy(), registry: registry.NewRegistry()}
}

func (handler *ProxyHandler) Handle(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	switch vars["method"] {
	case "assign":
		handler.proxyAssignHandle(w, r)
	case "dismiss":
		handler.proxyDismissHandle(w, r)
	case "TransferIODevice":
		handler.proxyTransferDeviceHandle(w, r)
	case "createOutput":
		handler.proxyCreateOutput(w, r)
	case "createInput":
		handler.proxyCreateInput(w, r)
	case "removeOutput":
		handler.proxyRemoveOutput(w, r)
	case "removeInput":
		handler.proxyRemoveInput(w, r)
	case "startOutput":
		handler.proxyStartOutput(w, r)
	case "startInput":
		handler.proxyStartInput(w, r)
	case "single":
		handler.proxySingle(w, r)
	default:
		http.Error(w, "wrong method route "+vars["method"], http.StatusMethodNotAllowed)
	}
}

func (handler *ProxyHandler) proxyAssignHandle(w http.ResponseWriter, r *http.Request) {
	var reqJSON requests.AssignRequest

	err := json.NewDecoder(r.Body).Decode(&reqJSON)
	if err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	err = handler._proxy.Assign(reqJSON.InputName, reqJSON.OutputName)
	if err != nil {
		http.Error(w, "assignment error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, "{\"success\":true}")
}

func (handler *ProxyHandler) proxyDismissHandle(w http.ResponseWriter, r *http.Request) {
}

func (handler *ProxyHandler) proxyTransferDeviceHandle(w http.ResponseWriter, r *http.Request) {
	var reqJSON requests.TransferIODeviceRequest
	err := json.NewDecoder(r.Body).Decode(&reqJSON)
	if err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	receiver := handler._proxy.GetInput(reqJSON.Input)
	if receiver == nil {
		http.Error(w, "receiver not found", http.StatusNotFound)
		return
	}
	sender := handler._proxy.GetOutput(reqJSON.Output)
	if sender == nil {
		http.Error(w, "sender not found", http.StatusNotFound)
		return
	}

	sender.SetDevice(receiver.GetDevice())
	fmt.Fprintf(w, "{\"success\":true}")
}

func (handler *ProxyHandler) proxyCreateOutput(w http.ResponseWriter, r *http.Request) {
	var reqJSON requests.CreateOutputRequest

	err := json.NewDecoder(r.Body).Decode(&reqJSON)
	if err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	sender, err := proxy.NewUDPSender(reqJSON.OutputName, reqJSON.Address)
	if err != nil {
		http.Error(w, "UDP Sender creation error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	handler._proxy.AddOutput(sender)
}

func (handler *ProxyHandler) proxyCreateInput(w http.ResponseWriter, r *http.Request) {
	var reqJSON requests.CreateInputRequest

	err := json.NewDecoder(r.Body).Decode(&reqJSON)
	if err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	switch reqJSON.Type {
	case "Astra":
		receiver, err := proxy.NewAstra(reqJSON.InputName, reqJSON.Login, reqJSON.Password, reqJSON.Modem)
		if err != nil {
			http.Error(w, "Astra input creation error: "+err.Error(), http.StatusInternalServerError)
		}
		handler._proxy.AddOutput(receiver)
		handler._proxy.AddInput(receiver)
	case "UDP":
		receiver, err := proxy.NewUDPReceiver(reqJSON.InputName, reqJSON.Address)
		if err != nil {
			http.Error(w, "UDP input creation error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		handler._proxy.AddInput(receiver)
	}
}

func (handler *ProxyHandler) proxyRemoveOutput(w http.ResponseWriter, r *http.Request) {
	var reqJSON requests.RemoveOutputRequest
	err := json.NewDecoder(r.Body).Decode(&reqJSON)
	if err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	err = handler._proxy.Dismiss(reqJSON.OutputName)
	if err != nil {
		http.Error(w, "Dismiss error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, "{\"success\":true}")
}

func (handler *ProxyHandler) proxyRemoveInput(w http.ResponseWriter, r *http.Request) {

}

func (handler *ProxyHandler) proxyStartOutput(w http.ResponseWriter, r *http.Request) {
	var reqJSON requests.StartOutputRequest

	err := json.NewDecoder(r.Body).Decode(&reqJSON)
	if err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	output := handler._proxy.GetOutput(reqJSON.OutputName)
	if output == nil {
		http.Error(w, "output not found", http.StatusNotFound)
		return
	}
	err = output.Activate()
	if err != nil {
		http.Error(w, "output activation error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, "{\"success\":true}")
	r.Body.Close()
}

func (handler *ProxyHandler) proxyStartInput(w http.ResponseWriter, r *http.Request) {
	var reqJSON requests.StartInputRequest

	err := json.NewDecoder(r.Body).Decode(&reqJSON)
	if err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	input := handler._proxy.GetInput(reqJSON.InputName)
	if input == nil {
		http.Error(w, "input not found", http.StatusNotFound)
		return
	}
	err = input.Activate()
	if err != nil {
		http.Error(w, "input activation error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, "{\"success\":true}")
	r.Body.Close()
}

func (handler *ProxyHandler) proxySingle(w http.ResponseWriter, r *http.Request) {
	var reqJSON requests.SingleRequest

	err := json.NewDecoder(r.Body).Decode(&reqJSON.IOs)
	if err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	for i := range reqJSON.IOs {
		ioVal := reqJSON.IOs[i]
		if (ioVal.Name != "") && (ioVal.Type != "") {
			switch ioVal.Type {
			case "InputUDP":
				recv, err := proxy.NewUDPReceiver(ioVal.Name, ioVal.Address)
				if err != nil {
					http.Error(w, "Creating InputUDP from single request can't be completed: "+err.Error(), http.StatusBadRequest)
					return
				}
				handler._proxy.AddInput(recv)
				if recv == nil {
					http.Error(w, "InputUDP didn't created due to server issue", http.StatusInternalServerError)
					return
				}
				if ioVal.Activate {
					err = recv.Activate()
					if err != nil {
						http.Error(w, "Activation "+ioVal.Name+" failed: "+err.Error(), http.StatusBadRequest)
						return
					}
				}
				if len(ioVal.TransferTo) > 0 {
					for ind := range ioVal.TransferTo {
						outp := handler._proxy.GetOutput(ioVal.TransferTo[ind])
						if outp == nil {
							http.Error(w, "Output "+ioVal.TransferTo[ind]+" doesn't exists", http.StatusBadRequest)
							return
						}
						outp.SetDevice(recv.GetDevice())
					}
				}
				if ioVal.Assign != nil {
					for ind := range ioVal.Assign {
						if ioVal.Assign[ind] != "" {
							err = handler._proxy.Assign(recv.GetName(), ioVal.Assign[ind])
							if err != nil {
								http.Error(w, "Assigning "+ioVal.Name+" to "+ioVal.Assign[ind]+" failed: "+err.Error(), http.StatusBadRequest)
								return
							}
						}
					}
				}

			case "OutputUDP":
				snd, err := proxy.NewUDPSender(ioVal.Name, ioVal.Address)
				if err != nil {
					http.Error(w, "Creating InputUDP from single request can't be completed: "+err.Error(), http.StatusBadRequest)
					return
				}
				handler._proxy.AddOutput(snd)
				if snd == nil {
					http.Error(w, "InputUDP didn't created due to server issue", http.StatusInternalServerError)
					return
				}
				if ioVal.Assign != nil {
					for ind := range ioVal.Assign {
						if ioVal.Assign[ind] != "" {
							err = handler._proxy.Assign(ioVal.Assign[ind], snd.GetName())
							if err != nil {
								http.Error(w, "Assigning "+ioVal.Name+" to "+ioVal.Assign[ind]+" failed: "+err.Error(), http.StatusBadRequest)
								return
							}
						}
					}
				}
				if ioVal.Activate {
					err = snd.Activate()
					if err != nil {
						http.Error(w, "Activation "+ioVal.Name+" failed: "+err.Error(), http.StatusBadRequest)
						return
					}
				}
			case "Astra":
				recvSnd, err := proxy.NewAstra(ioVal.Name, ioVal.Login, ioVal.Password, ioVal.AstraModemType)
				if err != nil {
					http.Error(w, "Creating InputUDP from single request can't be completed: "+err.Error(), http.StatusBadRequest)
					return
				}
				handler._proxy.AddInput(recvSnd)
				handler._proxy.AddOutput(recvSnd)
				if recvSnd == nil {
					http.Error(w, "InputUDP didn't created due to server issue", http.StatusInternalServerError)
					return
				}
				if ioVal.Activate {
					err = recvSnd.Activate()
					if err != nil {
						http.Error(w, "Activation "+ioVal.Name+" failed: "+err.Error(), http.StatusBadRequest)
						return
					}
				}
			}

		}
	}

	fmt.Fprintf(w, "{\"success\":true}")
	r.Body.Close()
}
