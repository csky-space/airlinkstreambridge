package receivers

type IReceiver interface {
	SetOnData(func(data []byte) error)
	//Open()
	//Stop()
}
