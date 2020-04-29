/*
 * Npcf_PolicyAuthorization Service API
 *
 * This is the Policy Authorization Service
 *
 * API version: 1.0.0
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package PolicyAuthorization

import (
	"free5gc/lib/http_wrapper"
	"free5gc/lib/openapi/models"
	"free5gc/src/pcf/logger"
	"free5gc/src/pcf/handler/message"
	"free5gc/src/pcf/pcf_util"

	"github.com/gin-gonic/gin"
)

// DeleteEventsSubsc - deletes the Events Subscription subresource
func DeleteEventsSubsc(c *gin.Context) {

	req := http_wrapper.NewRequest(c.Request, nil)
	req.Params["appSessionId"], _ = c.Params.Get("appSessionId")
	channelMsg := message.NewHttpChannelMessage(message.EventDeleteEventsSubsc, req)

	message.SendMessage(channelMsg)
	recvMsg := <-channelMsg.HttpChannel

	HTTPResponse := recvMsg.HTTPResponse
	c.JSON(HTTPResponse.Status, HTTPResponse.Body)
}

// UpdateEventsSubsc - creates or modifies an Events Subscription subresource
func UpdateEventsSubsc(c *gin.Context) {
	var eventsSubscReqData models.EventsSubscReqData
	err := c.ShouldBindJSON(&eventsSubscReqData)
	if err != nil {
		rsp := pcf_util.GetProblemDetail("Malformed request syntax", pcf_util.ERROR_REQUEST_PARAMETERS)
		logger.HandlerLog.Errorln(rsp.Detail)
		c.JSON(int(rsp.Status), rsp)
		return
	}
	if eventsSubscReqData.Events == nil || eventsSubscReqData.NotifUri == "" {
		rsp := pcf_util.GetProblemDetail("Errorneous/Missing Mandotory IE", pcf_util.ERROR_REQUEST_PARAMETERS)
		logger.HandlerLog.Errorln(rsp.Detail)
		c.JSON(int(rsp.Status), rsp)
		return
	}

	req := http_wrapper.NewRequest(c.Request, eventsSubscReqData)
	req.Params["appSessionId"], _ = c.Params.Get("appSessionId")
	channelMsg := message.NewHttpChannelMessage(message.EventUpdateEventsSubsc, req)

	message.SendMessage(channelMsg)
	recvMsg := <-channelMsg.HttpChannel

	HTTPResponse := recvMsg.HTTPResponse
	c.JSON(HTTPResponse.Status, HTTPResponse.Body)
}
