package main

import httpserver "myproject/http_server"

func main() {
	server := httpserver.NewHttpServer()
	server.Loop()
}
