/*
 * Npcf_BDTPolicyControl Service API
 *
 * The Npcf_BDTPolicyControl Service is used by an NF service consumer to retrieve background data transfer policies from the PCF and to update the PCF with the background data transfer policy selected by the NF service consumer.
 *
 * API version: 1.0.0
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package BDTPolicy

import (
	"gofree5gc/lib/http_wrapper"
	"gofree5gc/lib/openapi/models"
	"gofree5gc/src/pcf/pcf_handler/pcf_message"

	"github.com/gin-gonic/gin"
)

//CreateBDTPolicy - Create a new Individual BDT policy
func CreateBDTPolicy(c *gin.Context) {
	var bdtReqData models.BdtReqData
	c.ShouldBindJSON(&bdtReqData)

	req := http_wrapper.NewRequest(c.Request, bdtReqData)
	channelMsg := pcf_message.NewHttpChannelMessage(pcf_message.EventBDTPolicyCreate, req)

	pcf_message.SendMessage(channelMsg)
	recvMsg := <-channelMsg.HttpChannel
	HTTPResponse := recvMsg.HTTPResponse

	for key, val := range HTTPResponse.Header {
		c.Header(key, val[0])
	}

	c.JSON(HTTPResponse.Status, HTTPResponse.Body)
}
