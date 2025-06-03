package httpserver

import (
	"AirlinkStreamBridge/receivers"
	"AirlinkStreamBridge/senders"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
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
	Port     int    `json:"UDPPort"`
}

type DefaultReceiver struct {
	HostName  string `json:"hostName"`
	ModemName string `json:"modemName"`
	Password  string `json:"password"`
	UDPPort   int    `json:"UDPPort"`
}

type Http_server struct {
	router    *mux.Router
	tlsConfig *tls.Config
	cert      tls.Certificate
	wr        *receivers.WebrtcReceiver
	sender    senders.ISender
	listener  net.Listener

	closeMut       sync.Mutex
	shouldBeClosed bool
}

func rootHandle(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "You're at root")
}

func (server *Http_server) categoryHandle(w http.ResponseWriter, r *http.Request) {
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
		http.Error(w, "wrong category route "+vars["category"], http.StatusMethodNotAllowed)
	}
}

func (server *Http_server) webrtcCategoryHandle(w http.ResponseWriter, r *http.Request) {
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
		server.CloseHandle(w, r)
	case "createDefaultReceiver":
		server.createDefaultReceiverHandle(w, r)
	default:
		http.Error(w, "wrong method route "+vars["method"], http.StatusMethodNotAllowed)
	}
}

func (server *Http_server) videoCategoryHandle(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	switch vars["method"] {
	case "startVideo":
		server.startVideoHandle(w, r)
	case "stopVideo":
		server.stopVideoHandle(w, r)
	case "isRunning":
		server.videoIsRunningHandle(w, r)
	default:
		http.Error(w, "wrong method route "+vars["method"], http.StatusMethodNotAllowed)
	}
}

func (server *Http_server) connectionCategoryHandle(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	switch vars["method"] {
	case "open":
		server.openHandle(w, r)
	case "close":
		server.CloseHandle(w, r)
	case "isConnected":
		server.isConnected(w, r)
	case "closePeer":
		server.closePeerHandle(w, r)
	case "openPeer":
		server.openPeerHandle(w, r)
	default:
		http.Error(w, "wrong method route "+vars["method"], http.StatusMethodNotAllowed)
	}
}

func (server *Http_server) appCategoryHandle(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	switch vars["method"] {
	case "close":
		go server.CloseApp()
	default:
		http.Error(w, "wrong method route "+vars["method"], http.StatusMethodNotAllowed)
	}
}

func (server *Http_server) configureHandle(w http.ResponseWriter, r *http.Request) {
	if server.wr == nil {
		var reqJSON ConfigurationServerInput

		err := json.NewDecoder(r.Body).Decode(&reqJSON)
		if err != nil {
			http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
			return
		}

		server.wr.Configure(reqJSON.HostName, reqJSON.ModemName, reqJSON.Password)
	} else {
		http.Error(w, "webreceiver already opened. this call will be skip", http.StatusAlreadyReported)
	}

}

func (server *Http_server) createDefaultReceiverHandle(w http.ResponseWriter, r *http.Request) {
	log.Println("createDefaultReceiverHandle")
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

	if server.sender != nil {
		server.sender.Close()
	}
	server.sender, err = senders.NewUDPSender("", reqJSON.UDPPort)
	if err != nil {
		http.Error(w, "udp sender creation error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if server.wr != nil {
		server.wr.Close()
		server.wr = nil
	}
	server.createDefaultReceiver(reqJSON.HostName, reqJSON.ModemName, reqJSON.Password, w)
}

//func (server *Http_server) outputCategoryHandle(w http.ResponseWriter, r *http.Request) {
//	vars := mux.Vars(r)
//	switch vars["method"] {
//	case "setupProtocol":
//		server.setupOutputProtocolHandle(w, r)
//	default:
//		fmt.Fprintf(w, "{\"err\":\"wrong method route\"}")
//	}
//}

func (server *Http_server) setupCodecsHandle(w http.ResponseWriter, r *http.Request) {
	var codecs []receivers.JSONCodec

	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&codecs)
	if err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	err = server.wr.SetupCodecs(codecs)
	if err != nil {
		http.Error(w, "can't setup codecs: "+err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, "{\"success\":true}")
	r.Body.Close()
}

func (server *Http_server) setupOutputProtocolHandle(w http.ResponseWriter, r *http.Request) {
	log.Println("setupOutputProtocolHandle")
	var protocol OutputProtocol

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Failed to read body: %v", err)
		http.Error(w, "Failed to read body", http.StatusInternalServerError)
		return
	}
	r.Body.Close()

	err = json.Unmarshal(bodyBytes, &protocol)
	if err != nil {
		log.Printf("Invalid JSON: %v\n", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	log.Println("setupOutputProtocolHandle decoded")

	switch protocol.Protocol {
	case "UDP":
		log.Println("setup udp")
		var udpSetup UDPProtocol

		err = json.Unmarshal(bodyBytes, &udpSetup)
		if err != nil {
			log.Printf("Invalid JSON for UDP setup: %v", err)
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		if server.sender != nil {
			server.sender.Close()
		}
		server.sender, err = senders.NewUDPSender(udpSetup.Address, udpSetup.Port)
		if err != nil {
			http.Error(w, "udp sender creation error: "+err.Error(), http.StatusInternalServerError)
			fmt.Fprintf(w, "{\"error\":\"%v\"}", err)
		}

		fmt.Fprintf(w, "{\"success\":true}")
	default:
		http.Error(w, "unsupported protocol", http.StatusNotImplemented)
	}

}

func (server *Http_server) openHandle(w http.ResponseWriter, r *http.Request) {
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
		}()
	} else {
		fmt.Fprintf(w, `{"err":"webreceiver already opened. this call will be skip"}`)
	}
}

func (server *Http_server) CloseHandle(w http.ResponseWriter, r *http.Request) {
	if server.wr != nil {
		server.wr.Close()
	}
}

func (server *Http_server) closePeerHandle(w http.ResponseWriter, r *http.Request) {
	log.Println("closePeerHandle")
	if server.wr != nil {
		server.wr.PeerClose()
		fmt.Fprintf(w, `{"success":true}`)
	}
}

func (server *Http_server) openPeerHandle(w http.ResponseWriter, r *http.Request) {
	log.Println("openPeerHandle")
	if server.wr != nil {
		go server.wr.ReLaunchPeer()
		log.Println("http complete open")

		peerOpenedSubscriber := server.wr.PeerConnected.Subscribe()
		select {
		case <-peerOpenedSubscriber:
			log.Println("p open")
			fmt.Fprint(w, `{"success":true}`)
		case <-time.After(20 * time.Second):
			log.Println("timeout waiting for PeerConnected")
			fmt.Print(w, "\"warning\":\"Timeout waiting for connection. Not error! Webrtc would connect as soon as webrtc chain or airlink be restored\"")
		}
		server.wr.PeerConnected.Unsubscribe(peerOpenedSubscriber)
	} else {
		http.Error(w, "receiver not initialized", http.StatusConflict)
	}
}

type IsConnectedResponse struct {
	IsConnected bool `json:"isConnected"`
}

func (server *Http_server) isConnected(w http.ResponseWriter, r *http.Request) {
	if server.wr != nil {
		server.wr.IsConnected()
	}

	resp := IsConnectedResponse{
		IsConnected: server.wr.IsConnected(),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (server *Http_server) startVideoHandle(w http.ResponseWriter, r *http.Request) {
	if server.wr != nil {
		server.wr.SetTransmitEnabled(true)
		fmt.Fprintf(w, "{\"success\":true}")
		return
	}
	http.Error(w, "receiver doesn't exist", http.StatusConflict)
}

func (server *Http_server) stopVideoHandle(w http.ResponseWriter, r *http.Request) {
	if server.wr != nil {
		server.wr.SetTransmitEnabled(false)
		fmt.Fprintf(w, "{\"success\":true}")
		return
	}
	http.Error(w, "receiver doesn't exist", http.StatusConflict)
}

func (server *Http_server) videoIsRunningHandle(w http.ResponseWriter, r *http.Request) {
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

func (server *Http_server) setupTLSServer() {
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
		return
	}

	log.Println("HTTPS server has been started at https://localhost:8443")
}

func (server *Http_server) serve() {
	err := http.Serve(server.listener, server.router)
	if err != nil {
		log.Fatalf("HTTPS server error: %v", err)
	}
}

func (server *Http_server) CloseApp() {
	server.closeMut.Lock()
	defer server.closeMut.Unlock()
	server.shouldBeClosed = true
}

func NewHttpServer() *Http_server {
	server := &Http_server{}
	server.shouldBeClosed = false
	server.wr = nil
	return server
}

func (server *Http_server) Loop() {
	server.setupTLSServer()

	go server.serve()

	for !server.shouldBeClosed {
		time.Sleep(time.Second)
	}
}

func (server *Http_server) createDefaultReceiver(hostUrl string, login string, password string, w http.ResponseWriter) {
	var err error
	server.wr, err = receivers.NewDefaultWebrtcReceiver(hostUrl, login, password)
	if err != nil {
		log.Println("default receiver creation error "+err.Error(), http.StatusInternalServerError)
		http.Error(w, "default receiver creation error "+err.Error(), http.StatusInternalServerError)
		return
	}
	server.wr.SetOnData(func(data []byte) error {
		if server.sender != nil {
			err := server.sender.Send(data)
			if err != nil {
				server.sender.Close()
			}
			return err
		}
		return errors.New("sender doesn't exists")
	})
	log.Println("http complete creating")
	subscriber := server.wr.WebrtcReceiverCreated.Subscribe()
	select {
	case <-subscriber:
		log.Println("p open")
		fmt.Fprint(w, `{"success":true}`)
	case <-time.After(20 * time.Second):
		log.Println("timeout waiting for creating webrtc")
		http.Error(w, "webrtcreceiver creating timeout", http.StatusInternalServerError)
	}
	server.wr.WebrtcReceiverCreated.Unsubscribe(subscriber)
	fmt.Fprintf(w, "{\"success\":true}")
	log.Println("webrtc cl")
}
