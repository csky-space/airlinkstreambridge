package proxy

import (
	"errors"
)

type Proxy struct {
	Receiver
	inputs  map[string]IReceiver
	outputs map[string]ISender

	isTransparent bool
}

func NewProxy() *Proxy {
	return &Proxy{
		outputs: make(map[string]ISender),
		inputs:  make(map[string]IReceiver),
	}
}

func (proxy *Proxy) GetOutputs() map[string]ISender {
	return proxy.outputs
}

func (proxy *Proxy) GetInputs() map[string]IReceiver {
	return proxy.inputs
}

func (proxy *Proxy) AddOutput(sender ISender) {
	proxy.outputs[sender.GetName()] = sender
}

func (proxy *Proxy) AddInput(receiver IReceiver) {
	proxy.inputs[receiver.GetName()] = receiver
	if proxy.isTransparent {
		for _, value := range proxy.outputs {
			receiver.SubscribeOnData(value, func(data []byte) error {
				return proxy.Broadcast(data)
			})
		}

	}
}

func (proxy *Proxy) RemoveOutput(name string) {
	delete(proxy.outputs, name)
}

func (proxy *Proxy) RemoveInput(name string) {
	delete(proxy.inputs, name)
	if proxy.isTransparent {
		proxy.GetInput(name).UnsubscribeAll()
	}
}

func (proxy *Proxy) AssignExists(from string, to string) error {
	receiver := proxy.GetInput(from)
	sender := proxy.GetOutput(to)

	if receiver == nil {
		return errors.New("Receiver not found")
	}
	if sender == nil {
		return errors.New("Sender not found")
	}

	return proxy.Assign(from, to)
}

func (proxy *Proxy) DismissExists(proxyName string) error {
	if _, ok := proxy.outputs[proxyName]; !ok {
		return errors.New("Sender not found")
	}
	if _, ok := proxy.inputs[proxyName]; !ok {
		return errors.New("Receiver not found")
	}

	return proxy.Dismiss(proxyName)
}

func (proxy *Proxy) Assign(from string, to string) error {
	fromR := proxy.GetInput(from)
	toT := proxy.GetOutput(to)

	if (fromR != nil) && (toT != nil) {
		fromR.SubscribeOnData(toT, func(data []byte) error {
			return toT.Send(data)
		})
	}

	return nil
}

func (proxy *Proxy) Dismiss(proxyName string) error {
	delete(proxy.outputs, proxyName)
	delete(proxy.inputs, proxyName)
	return nil
}

func (proxy *Proxy) GetName() string {
	return "Proxy"
}

func (proxy *Proxy) GetOutput(name string) ISender {
	return proxy.outputs[name]
}

func (proxy *Proxy) GetInput(name string) IReceiver {
	return proxy.inputs[name]
}

func (proxy *Proxy) StartTransparent() {
	proxy.isTransparent = true
	for _, receiver := range proxy.GetInputs() {
		receiver.SubscribeOnData(proxy, func(data []byte) error {
			proxy.Broadcast(data)
			return nil
		})
	}
}

func (proxy *Proxy) StopTransparent() {
	proxy.isTransparent = false
	for _, receiver := range proxy.GetInputs() {
		receiver.UnsubscribeAll()
	}
}

func (proxy *Proxy) Broadcast(data []byte) error {
	for _, sender := range proxy.GetOutputs() {
		sender.Send(data)
	}
	return nil
}

func (proxy *Proxy) Send(data []byte, target string) error {
	sender := proxy.GetOutput(target)
	if sender != nil {
		return sender.Send(data)
	}
	return errors.New("Sender not found")
}
