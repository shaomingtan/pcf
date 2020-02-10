package Npcf_Callback

import (
	"gofree5gc/lib/http_wrapper"
	"gofree5gc/lib/openapi/models"
	"gofree5gc/src/pcf/pcf_handler/pcf_message"

	"github.com/gin-gonic/gin"
)

func AmfStatusChangeNotify(c *gin.Context) {
	amfStatusChangeNotification := models.AmfStatusChangeNotification{}
	req := http_wrapper.NewRequest(c.Request, amfStatusChangeNotification)
	channelMsg := pcf_message.NewHttpChannelMessage(pcf_message.EventAMFStatusChangeNotify, req)

	pcf_message.SendMessage(channelMsg)
	recvMsg := <-channelMsg.HttpChannel
	HTTPResponse := recvMsg.HTTPResponse
	c.JSON(HTTPResponse.Status, HTTPResponse.Body)
}
