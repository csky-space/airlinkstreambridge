package iceconfigurator

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"strconv"
	"time"
)

type PostResponse struct {
	AccessToken string `json:"accessToken"`
	Username    string `json:"username"`
}

type TurnServer struct {
	URL        string `json:"url"`
	Credential string `json:"credential"`
	Username   string `json:"username"`
}

type ICEConfigurator struct {
	customResolver  *net.Resolver
	customTransport *http.Transport
	customClient    *http.Client
	Token           string

	TurnServers     []TurnServer `json:"turnServers"`
	StunServers     []string     `json:"stunServers"`
	WsURL           string       `json:"wsUrl"`
	ModemVersion    int          `json:"modemVersion"`
	FirmwareVersion int          `json:"firmwareVersion"`
	Login           string       `json:"login"`
	FpvMode         bool         `json:"fpvMode"`
	FpvLatency      int          `json:"fpvLatency"`
}

func NewICEConfigurator(hostUrl string, login string, password string) (*ICEConfigurator, error) {

	ice := &ICEConfigurator{Login: login}

	ice.customResolver = &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := &net.Dialer{
				Timeout: time.Second * 2,
			}
			return d.DialContext(ctx, network, "8.8.8.8:53") // Google DNS
		},
	}

	ice.customTransport = &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
			Resolver:  ice.customResolver,
		}).DialContext,
	}
	ice.customTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	ice.customClient = &http.Client{
		Transport: ice.customTransport,
		Timeout:   10 * time.Second,
	}

	requestsPath := "https://" + hostUrl + "/api/groundStation"
	token, err := ice.getToken(login, password, requestsPath)
	ice.Token = token
	if err != nil {
		return nil, err
	}
	err = ice.getConfiguration(token, requestsPath)
	if err != nil {
		return nil, err
	}
	return ice, nil
}

func (ice *ICEConfigurator) getToken(login string, password string, requestsPath string) (string, error) {
	//log.Printf("login with: %s, %s, %s", login, password, requestsPath)
	loginData := map[string]interface{}{
		"name": login,
		"pass": password,
	}
	loginJsonData, err := json.Marshal(loginData)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", requestsPath+"/login", bytes.NewBuffer(loginJsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")

	loginResponse, err := ice.customClient.Do(req)
	if err != nil {
		return "", err
	}

	defer loginResponse.Body.Close()
	statusCode := loginResponse.StatusCode
	if statusCode >= 400 {
		if statusCode < 500 {
			return "", errors.New("client error. status code: " + strconv.Itoa(statusCode))
		} else {
			return "", errors.New("server error. status code: " + strconv.Itoa(statusCode))
		}

	}
	loginResponseBody, err := io.ReadAll(loginResponse.Body)
	if err != nil {
		return "", err
	}
	//log.Printf("login response body: %s", loginResponseBody)
	var accessToken PostResponse
	err = json.Unmarshal(loginResponseBody, &accessToken)
	if err != nil {
		return "", err
	}

	return accessToken.AccessToken, nil
}

func (ice *ICEConfigurator) getConfiguration(accessToken string, requestsPath string) error {
	req, err := http.NewRequest("GET", requestsPath+"/config", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := ice.customClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	configResponseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	err = json.Unmarshal(configResponseBody, &ice)
	if err != nil {
		return err
	}
	return nil
}
