package httpserver

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

type http_server struct {
	router    *mux.Router
	tlsConfig *tls.Config
	cert      tls.Certificate
}

func rootHandle(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "You're at root")
}

func categoryHandle(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	switch vars["category"] {
	case "VideoFlowControl":
		videoCategoryHandle(w, r)
	default:
		fmt.Fprintf(w, "{\"err\":\"wrong category route\"}")
	}
}

func videoCategoryHandle(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	switch vars["method"] {
	case "startVideo":
	case "stopVideo":
	default:
		fmt.Fprintf(w, "{\"err\":\"wrong method route\"}")
	}
}

func configurationServerCategoryHandle(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	switch vars["method"] {
	case "sendGroundStationCredentials":
	default:
		fmt.Fprintf(w, "{\"err\":\"wrong method route\"}")
	}
}

func startVideoHandle(w http.ResponseWriter, r *http.Request) {

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

func NewHttpServer() *http_server {
	server := &http_server{}

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
	server.router.HandleFunc("/{category}/{method}", categoryHandle).Methods("GET", "POST", "PUT")

	listener, err := tls.Listen("tcp", ":8443", server.tlsConfig)
	if err != nil {
		log.Fatalf("TLS listener creation error: %v", err)
	}

	log.Println("HTTPS server has been started at https://localhost:8443")
	go func() {
		err := http.Serve(listener, server.router)
		if err != nil {
			log.Fatalf("HTTPS server error: %v", err)
		}
	}()

	return server
}
