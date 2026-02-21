package registry

import (
	"AirlinkStreamBridge/proxy/UDP"
	"AirlinkStreamBridge/receivers"
	"reflect"
)

type Registry struct {
	types map[string]reflect.Type
}

func NewRegistry() *Registry {
	registry := &Registry{
		types: make(map[string]reflect.Type),
	}
	registry.RegisterType("Astra", reflect.TypeOf((*receivers.Astra)(nil)).Elem())
	registry.RegisterType("UDPReceiver", reflect.TypeOf((*UDP.UDPReceiver)(nil)).Elem())
	registry.RegisterType("UDPSender", reflect.TypeOf((*UDP.UDPSender)(nil)).Elem())
	registry.RegisterType("WebrtcReceiver", reflect.TypeOf((*receivers.WebrtcReceiver)(nil)).Elem())
	return registry
}

func (registry *Registry) RegisterType(name string, typ reflect.Type) {
	registry.types[name] = typ
}

func (registry *Registry) GetType(name string) (reflect.Type, bool) {
	typ, exists := registry.types[name]
	return typ, exists
}
