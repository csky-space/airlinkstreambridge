package httpserver

import (
	"AirlinkStreamBridge/receivers"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"log"
	"math/big"
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

type DefaultReceiver struct {
	HostName  string `json:"hostName"`
	ModemName string `json:"modemName"`
	Password  string `json:"password"`
	UDPPort   int    `json:"UDPPort"`
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
	log.Println("category handle")
	vars := mux.Vars(r)
	switch vars["category"] {
	case "Webrtc":
		server.webrtcCategoryHandle(w, r)
	case "Video":
		server.videoCategoryHandle(w, r)
	case "Connection":
		server.connectionCategoryHandle(w, r)
	case "App":
		server.appCategoryHandle(w, r)
	default:
		fmt.Fprintf(w, "{\"err\":\"wrong category route\"}")
	}
}

func (server *http_server) webrtcCategoryHandle(w http.ResponseWriter, r *http.Request) {
	log.Println("webrtc handle")
	vars := mux.Vars(r)
	switch vars["method"] {
	case "configure":
		server.configureHandle(w, r)
	case "setupOutputProtocol":
		server.setupOutputProtocolHandle(w, r)
	case "setupCodecs":
		server.setupCodecsHandle(w, r)
	case "open":
		server.openHandle(w, r)
	case "close":
		server.closeHandle(w, r)
	case "createDefaultReceiver":
		server.createDefaultReceiverHandle(w, r)
	default:
		fmt.Fprintf(w, "{\"err\":\"wrong method route\"}")
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
	case "closePeer":
		server.closePeerHandle(w, r)
	case "openPeer":
		server.openPeerHandle(w, r)
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

func (server *http_server) configureHandle(w http.ResponseWriter, r *http.Request) {
	if server.wr == nil {
		var reqJSON ConfigurationServerInput

		err := json.NewDecoder(r.Body).Decode(&reqJSON)
		if err != nil {
			http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
			return
		}

		server.wr.Configure(reqJSON.HostName, reqJSON.ModemName, reqJSON.Password)
	} else {
		fmt.Fprintf(w, "{\"err\":\"webreceiver already opened. this call will be skip\"}")
	}

}

func (server *http_server) createDefaultReceiverHandle(w http.ResponseWriter, r *http.Request) {
	log.Println("createDefaultReceiverHandle")
	if server.wr == nil || !server.wr.WsIsOpen() {
		log.Println("creating default")
		var reqJSON DefaultReceiver

		err := json.NewDecoder(r.Body).Decode(&reqJSON)
		if err != nil {
			http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		if server.wr != nil {
			server.wr.Close()
		}

		server.wr = receivers.NewDefaultWebrtcReceiver(reqJSON.HostName, reqJSON.ModemName, reqJSON.Password, reqJSON.UDPPort)
		fmt.Fprintf(w, "{\"success\":true}")
	} else {
		fmt.Fprintf(w, "{\"err\":\"webreceiver already opened. this call will be skip\"}")
	}

}

//func (server *http_server) outputCategoryHandle(w http.ResponseWriter, r *http.Request) {
//	vars := mux.Vars(r)
//	switch vars["method"] {
//	case "setupProtocol":
//		server.setupOutputProtocolHandle(w, r)
//	default:
//		fmt.Fprintf(w, "{\"err\":\"wrong method route\"}")
//	}
//}

func (server *http_server) setupCodecsHandle(w http.ResponseWriter, r *http.Request) {
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
			server.wr.Open()
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

func (server *http_server) closePeerHandle(w http.ResponseWriter, r *http.Request) {
	log.Println("closePeerHandle")
	if server.wr != nil {
		server.wr.PeerClose()
	}
}

func (server *http_server) openPeerHandle(w http.ResponseWriter, r *http.Request) {
	log.Println("openPeerHandle")
	if server.wr != nil {
		server.wr.ReLaunchPeer()
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

func generateSelfSignedCert() (certPEM, keyPEM []byte) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		log.Fatal(err)
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(365 * 24 * time.Hour)

	serialNumber, _ := rand.Int(rand.Reader, big.NewInt(1<<62))

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName: "localhost",
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,
		KeyUsage:  x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth,
		},
		BasicConstraintsValid: true,
		DNSNames:              []string{"localhost"},
		IPAddresses: []net.IP{
			net.ParseIP("127.0.0.1"),
			net.ParseIP("::1"),
		},
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		log.Fatal(err)
	}

	certPEMBlock := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	keyBytes, _ := x509.MarshalECPrivateKey(priv)
	keyPEMBlock := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBytes})

	return certPEMBlock, keyPEMBlock
}

func (server *http_server) setupTLSServer() {
	cert, key := generateSelfSignedCert()
	var err error
	server.cert, err = tls.X509KeyPair(cert, key)
	if err != nil {
		log.Fatal("failed to parse certificate: ", err)
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
	server.wr = nil
	return server
}

func (server *http_server) Loop() {
	server.setupTLSServer()

	go server.serve()

	for !server.shouldBeClosed {
		time.Sleep(time.Second)
	}
}
