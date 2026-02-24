package object

type Object struct {
	IObject
	name string
}

func NewObject(name string) *Object {
	return &Object{name: name}
}

func (o *Object) GetName() string {
	return o.name
}
