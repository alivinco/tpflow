package client

import (
	"encoding/json"
	log "github.com/Sirupsen/logrus"
	"github.com/alivinco/fimpgo"
	conmodel "github.com/alivinco/tpflow/connector/model"
	"github.com/alivinco/tpflow/flow"
	"github.com/alivinco/tpflow/model"
)

type ApiRemoteClient struct {
	sClient * fimpgo.SyncClient
	timeout int64
	instanceAddress string
	ctx *model.Context
}

func NewApiRemoteClient(sClient *fimpgo.SyncClient,instanceAddress string,ctx *model.Context) *ApiRemoteClient {
	sClient.AddSubscription("pt:j1/mt:evt/rt:app/rn:tpflow/ad:"+instanceAddress)
	return &ApiRemoteClient{sClient: sClient,instanceAddress:instanceAddress,timeout:15}
}

func (rc *ApiRemoteClient) GetListOfFlows()([]flow.FlowListItem,error) {
	reqMsg := fimpgo.NewNullMessage("cmd.flow.get_list","tpflow",nil,nil,nil)
	respMsg , err := rc.sClient.SendFimp("pt:j1/mt:cmd/rt:app/rn:tpflow/ad:"+rc.instanceAddress,reqMsg,rc.timeout)
	if err != nil {
		return nil,err
	}

	var resp []flow.FlowListItem
	err = json.Unmarshal(respMsg.GetRawObjectValue(), &resp)
	if err != nil {
		log.Error("Can't unmarshal ", err)
		return nil , err
	}
	return resp,nil

}

func (rc *ApiRemoteClient) GetFlowDefinition(flowId string) (*model.FlowMeta,error) {
	reqMsg := fimpgo.NewStringMessage("cmd.flow.get_definition","tpflow",flowId,nil,nil,nil)
	respMsg , err := rc.sClient.SendFimp("pt:j1/mt:cmd/rt:app/rn:tpflow/ad:"+rc.instanceAddress,reqMsg,rc.timeout)
	if err != nil {
		return nil,err
	}
	var resp model.FlowMeta
	err = json.Unmarshal(respMsg.GetRawObjectValue(), &resp)
	if err != nil {
		log.Error("Can't unmarshal ", err)
		return nil,err
	}
	return &resp,nil

}

func (rc *ApiRemoteClient) GetConnectorTemplate(templateId string) (conmodel.Instance,error) {
	var resp conmodel.Instance
	reqMsg := fimpgo.NewStringMessage("cmd.flow.get_connector_template","tpflow",templateId,nil,nil,nil)
	respMsg , err := rc.sClient.SendFimp("pt:j1/mt:cmd/rt:app/rn:tpflow/ad:"+rc.instanceAddress,reqMsg,rc.timeout)
	if err != nil {
		return resp,err
	}
	err = json.Unmarshal(respMsg.GetRawObjectValue(), &resp)
	if err != nil {
		log.Error("Can't unmarshal ", err)
		return resp,err
	}
	return resp,nil
}

//cmd.flow.get_connector_template

func (rc *ApiRemoteClient) GetConnectorPlugins() (map[string]conmodel.Plugin,error) {
	var resp map[string]conmodel.Plugin
	reqMsg := fimpgo.NewNullMessage("cmd.flow.get_connector_plugins","tpflow",nil,nil,nil)
	respMsg , err := rc.sClient.SendFimp("pt:j1/mt:cmd/rt:app/rn:tpflow/ad:"+rc.instanceAddress,reqMsg,rc.timeout)
	if err != nil {
		return resp,err
	}
	err = json.Unmarshal(respMsg.GetRawObjectValue(), &resp)
	if err != nil {
		log.Error("Can't unmarshal ", err)
		return resp,err
	}
	return resp,nil

}

func (rc *ApiRemoteClient) GetConnectorInstances() ([]conmodel.InstanceView,error) {
	var resp []conmodel.InstanceView
	reqMsg := fimpgo.NewNullMessage("cmd.flow.get_connector_instances","tpflow",nil,nil,nil)
	respMsg , err := rc.sClient.SendFimp("pt:j1/mt:cmd/rt:app/rn:tpflow/ad:"+rc.instanceAddress,reqMsg,rc.timeout)
	if err != nil {
		return resp,err
	}
	err = json.Unmarshal(respMsg.GetRawObjectValue(), &resp)
	if err != nil {
		log.Error("Can't unmarshal ", err)
		return resp,err
	}
	return resp,nil

}

func (rc *ApiRemoteClient) ImportFlow(flowDef []byte) (string, error) {
	//var resp []conmodel.InstanceView
	var flowDefJson interface{}
	err := json.Unmarshal(flowDef,&flowDefJson)
	if err != nil {
		log.Error("Can't unmarshal ", err)
		return "",err
	}
	reqMsg := fimpgo.NewMessage("cmd.flow.import","tpflow","object",flowDefJson,nil,nil,nil)
	respMsg , err := rc.sClient.SendFimp("pt:j1/mt:cmd/rt:app/rn:tpflow/ad:"+rc.instanceAddress,reqMsg,rc.timeout)
	if err != nil {
		return "",err
	}
	if err != nil {
		log.Error("Can't unmarshal ", err)
		return "",err
	}
	return respMsg.GetStringValue()
}

func (rc *ApiRemoteClient) UpdateFlowBin(flowDef []byte) (string, error) {
	var flowDefJson interface{}
	err := json.Unmarshal(flowDef,&flowDefJson)
	if err != nil {
		log.Error("Can't unmarshal ", err)
		return "",err
	}
	reqMsg := fimpgo.NewMessage("cmd.flow.update_definition","tpflow","object",flowDefJson,nil,nil,nil)
	respMsg , err := rc.sClient.SendFimp("pt:j1/mt:cmd/rt:app/rn:tpflow/ad:"+rc.instanceAddress,reqMsg,rc.timeout)
	if err != nil {
		return "",err
	}

	return respMsg.GetStringValue()
}

func (rc *ApiRemoteClient) ControlFlow(cmd string,id string) (string, error) {
	cmdVal := make(map[string]string)
	cmdVal["op"] = cmd
	cmdVal["id"] = id

	reqMsg := fimpgo.NewStrMapMessage("cmd.flow.ctrl","tpflow",cmdVal,nil,nil,nil)
	respMsg , err := rc.sClient.SendFimp("pt:j1/mt:cmd/rt:app/rn:tpflow/ad:"+rc.instanceAddress,reqMsg,rc.timeout)
	if err != nil {
		return "",err
	}
	return respMsg.GetStringValue()
}

func (rc *ApiRemoteClient) ImportFlowFromUrl(url string, token string) (string, error) {
	cmdVal := make(map[string]string)
	cmdVal["url"] = url
	cmdVal["token"] = token

	reqMsg := fimpgo.NewStrMapMessage("cmd.flow.import_from_url","tpflow",cmdVal,nil,nil,nil)
	respMsg , err := rc.sClient.SendFimp("pt:j1/mt:cmd/rt:app/rn:tpflow/ad:"+rc.instanceAddress,reqMsg,rc.timeout)
	if err != nil {
		return "",err
	}
	return respMsg.GetStringValue()
}


