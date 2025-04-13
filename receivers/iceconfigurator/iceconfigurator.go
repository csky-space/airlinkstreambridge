package iceconfigurator

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
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
	TurnServers     []TurnServer `json:"turnServers"`
	StunServers     []string     `json:"stunServers"`
	WsURL           string       `json:"wsUrl"`
	ModemVersion    int          `json:"modemVersion"`
	FirmwareVersion int          `json:"firmwareVersion"`
	Login           string       `json:"login"`
	FpvMode         bool         `json:"fpvMode"`
	FpvLatency      int          `json:"fpvLatency"`
}

func NewICEConfigurator(hostUrl string, login string, password string) *ICEConfigurator {
	ice := &ICEConfigurator{}
	requestsPath := "https://" + hostUrl + "/api/groundStation"

	ice.getConfiguration(ice.getToken(login, password, requestsPath), requestsPath)
	return ice
}

func (ice *ICEConfigurator) getToken(login string, password string, requestsPath string) string {
	log.Printf("login with: %s, %s, %s", login, password, requestsPath)
	loginData := map[string]interface{}{
		"name": login,
		"pass": password,
	}
	loginJsonData, err := json.Marshal(loginData)
	if err != nil {
		panic(err)
	}

	loginResponse, err := http.Post(requestsPath+"/login", "application/json", bytes.NewBuffer(loginJsonData))
	if err != nil {
		panic(err)
	}
	defer loginResponse.Body.Close()

	loginResponseBody, err := io.ReadAll(loginResponse.Body)
	if err != nil {
		panic(err)
	}
	var accessToken PostResponse
	err = json.Unmarshal(loginResponseBody, &accessToken)
	if err != nil {
		panic(err)
	}

	return accessToken.AccessToken
}

func (ice *ICEConfigurator) getConfiguration(accessToken string, requestsPath string) {
	req, err := http.NewRequest("GET", requestsPath+"/config", nil)
	if err != nil {
		panic(err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	configResponseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(configResponseBody, &ice)
	if err != nil {
		panic(err)
	}
}
