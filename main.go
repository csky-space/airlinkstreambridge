package main

import (
	"fmt"
	"myproject/receivers"
	"time"
)

func main() {
	wr := receivers.NewWebrtcReceiver("stage.air-link.space", "001D0", "HM9-qc5-Ddq-mgy")

	for wr.WsIsOpen() {
		fmt.Println("ws is open")
		time.Sleep(time.Millisecond * 1000)
	}
}
