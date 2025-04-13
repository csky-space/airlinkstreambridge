package httpserver

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"log"
	"math/big"
	"myproject/receivers"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
)

type ConfigurationServerInput struct {
	HostName  string `json:"hostName"`
	ModemName string `json:"modemName"`
	Password  string `json:"password"`
}

type OutputProtocol struct {
	Protocol string `json:"protocol"`
}

type UDPProtocol struct {
	Protocol string `json:"protocol"`
	Address  string `json:"address"`
	Port     int    `json:"port"`
}

type http_server struct {
	router    *mux.Router
	tlsConfig *tls.Config
	cert      tls.Certificate
	wr        *receivers.WebrtcReceiver
	listener  net.Listener

	closeMut       sync.Mutex
	shouldBeClosed bool
}

func rootHandle(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "You're at root")
}

func (server *http_server) categoryHandle(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	switch vars["category"] {
	case "VideoFlowControl":
		server.videoCategoryHandle(w, r)
	case "Connection":
		server.connectionCategoryHandle(w, r)
	case "App":
		server.appCategoryHandle(w, r)
	case "Output":
		server.outputCategoryHandle(w, r)
	case "MediaSetup":
		server.setupMediaCategory(w, r)
	default:
		fmt.Fprintf(w, "{\"err\":\"wrong category route\"}")
	}
}

func (server *http_server) videoCategoryHandle(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	switch vars["method"] {
	case "startVideo":
		server.startVideoHandle(w, r)
	case "stopVideo":
		server.stopVideoHandle(w, r)
	case "isRunning":
		server.videoIsRunningHandle(w, r)
	default:
		fmt.Fprintf(w, "{\"err\":\"wrong method route\"}")
	}
}

func (server *http_server) connectionCategoryHandle(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	switch vars["method"] {
	case "open":
		server.openHandle(w, r)
	case "close":
		server.closeHandle(w, r)
	case "isConnected":
		server.isConnected(w, r)
	default:
		fmt.Fprintf(w, "{\"err\":\"wrong method route\"}")
	}
}

func (server *http_server) appCategoryHandle(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	switch vars["method"] {
	case "close":
		server.closeApp()
	default:
		fmt.Fprintf(w, "{\"err\":\"wrong method route\"}")
	}
}

func (server *http_server) outputCategoryHandle(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	switch vars["method"] {
	case "setupProtocol":
		server.setupOutputProtocolHandle(w, r)
	default:
		fmt.Fprintf(w, "{\"err\":\"wrong method route\"}")
	}
}

func (server *http_server) setupMediaCategory(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	switch vars["method"] {
	case "setupCodecs":
		server.setupCodecs(w, r)
	default:
		fmt.Fprintf(w, "{\"err\":\"wrong method route\"}")
	}
}

func (server *http_server) setupCodecs(w http.ResponseWriter, r *http.Request) {
	var codecs []receivers.JSONCodec

	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&codecs)
	if err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	server.wr.SetupCodecs(codecs)
	r.Body.Close()
}

func (server *http_server) setupOutputProtocolHandle(w http.ResponseWriter, r *http.Request) {
	var protocol OutputProtocol

	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&protocol)
	if err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	switch protocol.Protocol {
	case "UDP":
		var udpSetup UDPProtocol
		udpErr := decoder.Decode(&udpSetup)
		if udpErr != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		server.wr.SetupUDP(udpSetup.Address, udpSetup.Port)
	default:
		fmt.Fprintf(w, "{\"err\":\"wrong method route\"}")

	}
	r.Body.Close()
}

func (server *http_server) openHandle(w http.ResponseWriter, r *http.Request) {
	if server.wr == nil {
		var reqJSON ConfigurationServerInput

		err := json.NewDecoder(r.Body).Decode(&reqJSON)
		if err != nil {
			http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
			return
		}

		go func() {
			server.wr = receivers.NewWebrtcReceiver(reqJSON.HostName, reqJSON.ModemName, reqJSON.Password)
			defer server.wr.Close()
			for server.wr.WsIsOpen() {
				fmt.Println("ws is open")
				time.Sleep(time.Millisecond * 1000)
			}
			server.wr = nil
		}()
	} else {
		fmt.Fprintf(w, "{\"err\":\"webreceiver already opened. this call will be skip\"}")
	}
}

func (server *http_server) closeHandle(w http.ResponseWriter, r *http.Request) {
	if server.wr != nil {
		server.wr.Close()
		server.wr = nil
	}
}

type IsConnectedResponse struct {
	IsConnected bool `json:"isConnected"`
}

func (server *http_server) isConnected(w http.ResponseWriter, r *http.Request) {
	if server.wr != nil {
		server.wr.IsConnected()
	}

	resp := IsConnectedResponse{
		IsConnected: server.wr.IsConnected(),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (server *http_server) startVideoHandle(w http.ResponseWriter, r *http.Request) {
	if server.wr != nil {
		server.wr.SetTransmitEnabled(true)
	}
}

func (server *http_server) stopVideoHandle(w http.ResponseWriter, r *http.Request) {
	if server.wr != nil {
		server.wr.SetTransmitEnabled(false)
	}
}

func (server *http_server) videoIsRunningHandle(w http.ResponseWriter, r *http.Request) {
	if server.wr != nil {
		server.wr.IsConnected()
	}

	resp := IsConnectedResponse{
		IsConnected: server.wr.VideoIsRunning(),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func generateSelfSignedCert() (tls.Certificate, error) {
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return tls.Certificate{}, err
	}

	serialNumber, err := rand.Int(rand.Reader, big.NewInt(1<<62))
	if err != nil {
		return tls.Certificate{}, err
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName: "localhost",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{"localhost"},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privKey.PublicKey, privKey)
	if err != nil {
		return tls.Certificate{}, err
	}

	keyPEM := pem.EncodeToMemory(
		&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privKey)},
	)

	certPEM := pem.EncodeToMemory(
		&pem.Block{Type: "CERTIFICATE", Bytes: certDER},
	)

	return tls.X509KeyPair(certPEM, keyPEM)
}

func (server *http_server) setupTLSServer() {
	var err error
	server.cert, err = generateSelfSignedCert()
	if err != nil {
		log.Fatalf("Certificate generation error: %v", err)
	}

	server.tlsConfig = &tls.Config{
		Certificates: []tls.Certificate{server.cert},
		MinVersion:   tls.VersionTLS12,
	}

	server.router = mux.NewRouter()
	server.router.HandleFunc("/", rootHandle).Methods("GET")
	server.router.HandleFunc("/{category}/{method}", server.categoryHandle).Methods("GET", "POST", "PUT")

	server.listener, err = tls.Listen("tcp", ":8443", server.tlsConfig)
	if err != nil {
		log.Fatalf("TLS listener creation error: %v", err)
	}

	log.Println("HTTPS server has been started at https://localhost:8443")
}

func (server *http_server) serve() {
	err := http.Serve(server.listener, server.router)
	if err != nil {
		log.Fatalf("HTTPS server error: %v", err)
	}
}

func (server *http_server) closeApp() {
	server.closeMut.Lock()
	defer server.closeMut.Unlock()
	server.shouldBeClosed = true
}

func NewHttpServer() *http_server {
	server := &http_server{}
	server.shouldBeClosed = false
	return server
}

func (server *http_server) Loop() {
	server.setupTLSServer()

	go server.serve()

	for !server.shouldBeClosed {
		time.Sleep(time.Second)
	}
}
