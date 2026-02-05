package registry

import (
	"AirlinkStreamBridge/proxy"
	"reflect"
)

type Registry struct {
	types map[string]reflect.Type
}

func NewRegistry() *Registry {
	registry := &Registry{
		types: make(map[string]reflect.Type),
	}
	registry.RegisterType("Astra", reflect.TypeOf((*proxy.Astra)(nil)).Elem())
	registry.RegisterType("UDPReceiver", reflect.TypeOf((*proxy.UDPReceiver)(nil)).Elem())
	registry.RegisterType("UDPSender", reflect.TypeOf((*proxy.UDPSender)(nil)).Elem())
	registry.RegisterType("WebrtcReceiver", reflect.TypeOf((*proxy.WebrtcReceiver)(nil)).Elem())
	return registry
}

func (registry *Registry) RegisterType(name string, typ reflect.Type) {
	registry.types[name] = typ
}

func (registry *Registry) GetType(name string) (reflect.Type, bool) {
	typ, exists := registry.types[name]
	return typ, exists
}
