package agent

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/EdgeMatrixChain/edge-matrix-core/core/helper/rpc"
)

type AuthAgent struct {
	appPath    string
	httpClient *rpc.FastHttpClient
}

func NewAuthAgent(appPath string) *AuthAgent {
	return &AuthAgent{
		httpClient: rpc.NewDefaultHttpClient(),
		appPath:    appPath,
	}
}

type GetAuthDataResponse struct {
	Result int      `json:"_result"`
	Desc   string   `json:"_desc"`
	Data   AuthData `json:"data"`
}

type AuthData struct {
	ApiToken string `json:"apiToken"`
	Result   bool   `json:"result"`
}

func (p *AuthAgent) AuthBearer(apiKey string, nodeId string, port int) (result bool, apiToken string, err error) {
	err = nil
	result = false
	apiUrl := p.appPath + "/openapi/task/checkApikey"
	data := `{"apikey":"%s", "nodeId":"%s", "port":"%d"}`
	postJson := fmt.Sprintf(data, apiKey, nodeId, port)
	jsonBytes, err := p.httpClient.SendPostJsonRequest(apiUrl, []byte(postJson))
	if err != nil {
		err = errors.New("AuthBearer error:" + err.Error())
		return
	}
	response := &GetAuthDataResponse{}
	err = json.Unmarshal(jsonBytes, response)
	if err != nil {
		err = errors.New("GetAuthDataResponse json.Unmarshal error")
		return
	}

	if response.Result != 0 {
		err = errors.New("AuthBearer error:" + response.Desc)
		result = false
		apiToken = ""
		return
	}
	if !response.Data.Result {
		result = false
		apiToken = ""
		return
	}
	result = true
	apiToken = response.Data.ApiToken
	return
}
