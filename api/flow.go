package api

import (
	"encoding/json"
	"github.com/alivinco/fimpgo"
	"github.com/alivinco/tpflow/connector/plugins"
	"github.com/alivinco/tpflow/flow"
	"github.com/alivinco/tpflow/model"
	"github.com/labstack/echo"
	log "github.com/Sirupsen/logrus"
	"io/ioutil"
	"net/http"
)

type FlowApi struct {
	flowManager *flow.Manager
	echo        *echo.Echo
	msgTransport *fimpgo.MqttTransport
}

func NewFlowApi(flowManager *flow.Manager, echo *echo.Echo) *FlowApi {
	ctxApi := FlowApi{flowManager: flowManager, echo: echo}
	//ctxApi.RegisterRestApi()
	return &ctxApi
}

func (ctx *FlowApi) RegisterRestApi() {
	ctx.echo.GET("/fimp/flow/list", func(c echo.Context) error {
		resp := ctx.flowManager.GetFlowList()
		return c.JSON(http.StatusOK, resp)
	})
	ctx.echo.GET("/fimp/flow/definition/:id", func(c echo.Context) error {
		id := c.Param("id")
		var resp *model.FlowMeta
		if id == "-" {
			flow := ctx.flowManager.GenerateNewFlow()
			resp = &flow
		} else {
			resp = ctx.flowManager.GetFlowById(id).FlowMeta
		}

		return c.JSON(http.StatusOK, resp)
	})

	ctx.echo.GET("/fimp/connector/template/:id", func(c echo.Context) error {
		id := c.Param("id")
		result := plugins.GetConfigurationTemplate(id)
		return c.JSON(http.StatusOK, result)
	})

	ctx.echo.GET("/fimp/connector/plugins", func(c echo.Context) error {
		result := plugins.GetPlugins()
		return c.JSON(http.StatusOK, result)
	})

	ctx.echo.GET("/fimp/connector/list", func(c echo.Context) error {
		result := ctx.flowManager.GetConnectorRegistry().GetAllInstances()
		return c.JSON(http.StatusOK, result)
	})

	ctx.echo.POST("/fimp/flow/definition/:id", func(c echo.Context) error {
		id := c.Param("id")
		body, err := ioutil.ReadAll(c.Request().Body)
		if err != nil {
			return err
		}
		ctx.flowManager.UpdateFlowFromJsonAndSaveToStorage(id, body)
		return c.NoContent(http.StatusOK)
	})

	ctx.echo.PUT("/fimp/flow/definition/import", func(c echo.Context) error {
		body, err := ioutil.ReadAll(c.Request().Body)
		if err != nil {
			return err
		}
		ctx.flowManager.ImportFlow(body)
		return c.NoContent(http.StatusOK)
	})

	ctx.echo.PUT("/fimp/flow/definition/import_from_url", func(c echo.Context) error {

		body, err := ioutil.ReadAll(c.Request().Body)
		if err != nil {
			return err
		}
		request := ImportFlowFromUrlRequest{}
		err = json.Unmarshal(body, &request)
		if err != nil {
			log.Error("Can't parse request ", err)
		}

		// Get the data
		resp, err := http.Get(request.Url)
		if err != nil {
			return err
		}
		flow, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Error("Can't read file from url ", err)
			return err
		}
		log.Info("Importing flow")
		ctx.flowManager.ImportFlow(flow)
		return c.NoContent(http.StatusOK)
	})

	ctx.echo.POST("/fimp/flow/ctrl/:id/:op", func(c echo.Context) error {
		id := c.Param("id")
		op := c.Param("op")

		switch op {
		case "send-inclusion-report":
			ctx.flowManager.GetFlowById(id).SendInclusionReport()
		case "send-exclusion-report":
			ctx.flowManager.GetFlowById(id).SendExclusionReport()
		case "start":
			ctx.flowManager.ControlFlow("START", id)
		case "stop":
			ctx.flowManager.ControlFlow("STOP", id)

		}

		return c.NoContent(http.StatusOK)
	})

	ctx.echo.DELETE("/fimp/flow/definition/:id", func(c echo.Context) error {
		id := c.Param("id")
		ctx.flowManager.DeleteFlowFromStorage(id)
		return c.NoContent(http.StatusOK)
	})

}

func (ctx *FlowApi) RegisterMqttApi(msgTransport *fimpgo.MqttTransport) {
	ctx.msgTransport = msgTransport
	// TODO : Implement dynamic addressing and discovery
	ctx.msgTransport.Subscribe("pt:j1/mt:cmd/rt:app/rn:tpflow/ad:1")
	apiCh := make(fimpgo.MessageCh, 10)
	ctx.msgTransport.RegisterChannel("flow-api",apiCh)
	var fimp *fimpgo.FimpMessage
	go func() {
		for {

			newMsg := <-apiCh
			fimp = nil
			log.Debug("New flow message of type ", newMsg.Payload.Type)
			switch newMsg.Payload.Type {
			case "cmd.flow.get_list":
				val := ctx.flowManager.GetFlowList()
				fimp = fimpgo.NewMessage("evt.flow.list_report", "tpflow", "object", val, nil, nil, newMsg.Payload)

			case "cmd.flow.get_definition":
				var resp *model.FlowMeta
				id, _ := newMsg.Payload.GetStringValue()
				if id == "-" {
					flow := ctx.flowManager.GenerateNewFlow()
					resp = &flow
				} else {
					resp = ctx.flowManager.GetFlowById(id).FlowMeta
				}
				fimp = fimpgo.NewMessage("evt.flow.definition_report", "tpflow", "object", resp, nil, nil, newMsg.Payload)

			case "cmd.flow.get_connector_template":
				id, _ := newMsg.Payload.GetStringValue()
				resp := plugins.GetConfigurationTemplate(id)
				fimp = fimpgo.NewMessage("cmd.flow.connector_template_report", "tpflow", "object", resp, nil, nil, newMsg.Payload)

			case "cmd.flow.get_connector_plugins":
				resp := plugins.GetPlugins()
				fimp = fimpgo.NewMessage("evt.flow.connector_plugins_report", "tpflow", "object", resp, nil, nil, newMsg.Payload)

			case "cmd.flow.get_connector_instances":
				resp := ctx.flowManager.GetConnectorRegistry().GetAllInstances()
				fimp = fimpgo.NewMessage("evt.flow.connector_instances_report", "tpflow", "object", resp, nil, nil, newMsg.Payload)

			case "cmd.flow.update_definition":
				flowMeta := model.FlowMeta{}
				flowJsonDef := newMsg.Payload.GetRawObjectValue()
				err := json.Unmarshal(flowJsonDef, &flowMeta)
				if err != nil {
					log.Error("<FlMan> Can't unmarshel flow definition.")
					fimp = fimpgo.NewMessage("evt.flow.update_report", "tpflow", "string", err, nil, nil, newMsg.Payload)
					break
				}
				ctx.flowManager.UpdateFlowFromJsonAndSaveToStorage(flowMeta.Id, flowJsonDef)
				fimp = fimpgo.NewMessage("evt.flow.update_report", "tpflow", "string", "ok", nil, nil, newMsg.Payload)

			case "cmd.flow.import":
				resp := "ok"
				err := ctx.flowManager.ImportFlow(newMsg.Payload.GetRawObjectValue())
				if err != nil {
					resp = err.Error()
				}
				fimp = fimpgo.NewMessage("evt.flow.import_report", "tpflow", "string", resp, nil, nil, newMsg.Payload)

			case "cmd.flow.ctrl":
				resp := "ok"

				val, err := newMsg.Payload.GetStrMapValue()
				if err != nil {
					log.Error("Wrong value format ")
					fimp = fimpgo.NewMessage("evt.flow.ctrl_report", "tpflow", "string", err.Error(), nil, nil, newMsg.Payload)
					break

				}
				op, ok1 := val["op"]
				id, ok2 := val["id"]

				if !ok1 || !ok2 {
					fimp = fimpgo.NewMessage("evt.flow.ctrl_report", "tpflow", "string", "missing param", nil, nil, newMsg.Payload)
					break
				}
				switch op {
				case "send-inclusion-report":
					ctx.flowManager.GetFlowById(id).SendInclusionReport()
				case "send-exclusion-report":
					ctx.flowManager.GetFlowById(id).SendExclusionReport()
				case "start":
					err = ctx.flowManager.ControlFlow("START", id)
				case "stop":
					err = ctx.flowManager.ControlFlow("STOP", id)

				}
				if err != nil {
					resp = err.Error()
				}
				fimp = fimpgo.NewMessage("evt.flow.ctr_report", "tpflow", "string", resp, nil, nil, newMsg.Payload)

			case "cmd.flow.delete":
				resp := "ok"
				id,err := newMsg.Payload.GetStringValue()
				if err == nil {
					ctx.flowManager.DeleteFlowFromStorage(id)
				}else {
					resp = err.Error()
				}
				fimp = fimpgo.NewMessage("evt.flow.delete_report", "tpflow", "string", resp, nil, nil, newMsg.Payload)

			case "cmd.flow.import_from_url":
				resp := "ok"
				val, err := newMsg.Payload.GetStrMapValue()
				if err != nil {
					log.Error("Wrong value format ")
					fimp = fimpgo.NewMessage("evt.flow.import_report", "tpflow", "string", err.Error(), nil, nil, newMsg.Payload)
					break

				}
				url, ok := val["url"]
				if !ok {
					log.Error("Url is not defined ")
					fimp = fimpgo.NewMessage("evt.flow.import_report", "tpflow", "string", err.Error(), nil, nil, newMsg.Payload)
					break
				}
				// Get the data
				hresponse, err := http.Get(url)
				//token ,ok := val["token"]
				if err != nil {
					log.Error("Can't load file from url , error = ", err)
					fimp = fimpgo.NewMessage("evt.flow.import_report", "tpflow", "string", err.Error(), nil, nil, newMsg.Payload)
					break
				}
				bflow, err := ioutil.ReadAll(hresponse.Body)
				if err != nil {
					log.Error("Can't read file from url ", err)
					fimp = fimpgo.NewMessage("evt.flow.import_report", "tpflow", "string", err.Error(), nil, nil, newMsg.Payload)
					break
				}
				log.Info("Importing flow")
				resp = "ok"
				if err := ctx.flowManager.ImportFlow(bflow); err != nil {
					resp = err.Error()
				}
				fimp = fimpgo.NewMessage("evt.flow.import_report", "tpflow", "string", resp, nil, nil, newMsg.Payload)

			}
			if fimp != nil {
				addr := fimpgo.Address{MsgType: fimpgo.MsgTypeEvt, ResourceType: fimpgo.ResourceTypeApp, ResourceName: "tpflow", ResourceAddress: "1",}
				ctx.msgTransport.Publish(&addr, fimp)
			}

		}
	}()


}

type ImportFlowFromUrlRequest struct {
	Url   string
	Token string
}
