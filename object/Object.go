package object

type Object struct {
	name string
}

func NewObject(name string) *Object {
	return &Object{name: name}
}
