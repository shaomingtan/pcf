package Npcf_Callback

import (
	"github.com/gin-gonic/gin"
	"gofree5gc/lib/http_wrapper"
	"gofree5gc/lib/openapi/models"
	"gofree5gc/src/pcf/logger"
	"gofree5gc/src/pcf/pcf_handler/pcf_message"
	"net/http"
)

func AmfStatusChangeNotify(c *gin.Context) {
	var request models.AmfStatusChangeNotification

	err := c.ShouldBindJSON(&request)
	if err != nil {
		problemDetail := "[Request Body] " + err.Error()
		rsp := models.ProblemDetails{
			Title:  "Malformed request syntax",
			Status: http.StatusBadRequest,
			Detail: problemDetail,
		}
		logger.CallbackLog.Errorln(problemDetail)
		c.JSON(http.StatusBadRequest, rsp)
		return
	}

	req := http_wrapper.NewRequest(c.Request, request)

	channelMsg := pcf_message.NewHttpChannelMessage(pcf_message.EventAMFStatusChangeNotify, req)
	pcf_message.SendMessage(channelMsg)

	recvMsg := <-channelMsg.HttpChannel
	HTTPResponse := recvMsg.HTTPResponse
	c.JSON(HTTPResponse.Status, HTTPResponse.Body)
}
