package agent

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/emc-protocol/edge-matrix-core/core/helper/rpc"
)

type AppAgent struct {
	appPath    string
	httpClient *rpc.FastHttpClient
}

func NewAppAgent(appPath string) *AppAgent {
	return &AppAgent{
		httpClient: rpc.NewDefaultHttpClient(),
		appPath:    appPath,
	}
}

type GetDataResponse struct {
	Data string `json:"data"`
}

type GetBoolResponse struct {
	result bool `json:"result"`
}

func (p *AppAgent) BindAppNode(nodeId string) (err error) {
	err = nil
	apiUrl := p.appPath + "/hubapi/v1/bindNode"
	bindReq := `{"nodeId":"%s"}`
	postJson := fmt.Sprintf(bindReq, nodeId)
	_, err = p.httpClient.SendPostJsonRequest(apiUrl, []byte(postJson))
	if err != nil {
		err = errors.New("BindNode error:" + err.Error())
		return
	}
	return
}

func (p *AppAgent) ValidateApiKey(apiKey string) (result bool, err error) {
	err = nil
	result = false
	apiUrl := p.appPath + "/hubapi/v1/validateApiKey"
	data := `{"apiKey":"%s"}`
	postJson := fmt.Sprintf(data, apiKey)
	jsonBytes, err := p.httpClient.SendPostJsonRequest(apiUrl, []byte(postJson))
	if err != nil {
		err = errors.New("ValidateApiKey error:" + err.Error())
		return
	}
	response := &GetBoolResponse{}
	err = json.Unmarshal(jsonBytes, response)
	if err != nil {
		err = errors.New("GetBoolResponse json.Unmarshal error")
		return
	}
	result = response.result
	return
}

func (p *AppAgent) GetProxyPath() (err error, proxyPath string) {
	err = nil
	proxyPath = "/hubapi/v1/proxy"
	return
}

func (p *AppAgent) GetAppNode() (err error, nodeId string) {
	err = nil
	nodeId = ""
	apiUrl := p.appPath + "/hubapi/v1/getNode"
	jsonBytes, err := p.httpClient.SendGetRequest(apiUrl)
	if err != nil {
		err = errors.New("GetAppNode error:" + err.Error())
		return
	}
	response := &GetDataResponse{}
	err = json.Unmarshal(jsonBytes, response)
	if err != nil {
		err = errors.New("GetDataResponse json.Unmarshal error")
		return
	}
	nodeId = response.Data
	return
}

func (p *AppAgent) GetAppOrigin() (err error, appOrigin string) {
	err = nil
	appOrigin = ""
	apiUrl := p.appPath + "/hubapi/v1/getOrigin"
	jsonBytes, err := p.httpClient.SendGetRequest(apiUrl)
	if err != nil {
		err = errors.New("GetAppOrigin error:" + err.Error())
		return
	}
	response := &GetDataResponse{}
	err = json.Unmarshal(jsonBytes, response)
	if err != nil {
		err = errors.New("GetAppOriginResponse json.Unmarshal error")
		return
	}
	appOrigin = response.Data
	return
}

func (p *AppAgent) GetAppIdl() (err error, appOrigin string) {
	err = nil
	appOrigin = ""
	apiUrl := p.appPath + "/hubapi/v1/getIdl"
	jsonBytes, err := p.httpClient.SendGetRequest(apiUrl)
	if err != nil {
		err = errors.New("GetAppIdl error:" + err.Error())
		return
	}
	response := &GetDataResponse{}
	err = json.Unmarshal(jsonBytes, response)
	if err != nil {
		err = errors.New("GetAppOriginResponse json.Unmarshal error")
		return
	}
	appOrigin = response.Data
	return
}
