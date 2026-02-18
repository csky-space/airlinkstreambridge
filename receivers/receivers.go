package receivers

type Receivers struct {
	data map[string]IReceiver
}

func (recvs *Receivers) GetReceivers() map[string]IReceiver {
	return recvs.data
}

//func (recvs *Receivers)
