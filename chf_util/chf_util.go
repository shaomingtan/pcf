package chf_util

import (
	"context"
	"net/http"

	"github.com/free5gc/pcf/consumer"
	pcf_context "github.com/free5gc/pcf/context"
	"github.com/free5gc/pcf/factory"
	"github.com/free5gc/pcf/logger"
	"github.com/free5gc/pcf/util"

	"github.com/free5gc/openapi/Nnrf_NFDiscovery"
	"github.com/free5gc/openapi/models"
)

// getCHFUri will return the first CHF uri retrived from NRF
func getCHFUri() string {
	logger.ExtensionLog.Infof("Find available CHF Servers\n")

	NrfUri := pcf_context.PCF_Self().NrfUri
	localVarOptionals := Nnrf_NFDiscovery.SearchNFInstancesParamOpts{}

	resp, err := consumer.SendSearchNFInstances(NrfUri, models.NfType_CHF, models.NfType_PCF, localVarOptionals)
	if err != nil {
		logger.ExtensionLog.Error(err.Error())
		return ""
	}

	for _, nfProfile := range resp.NfInstances {
		uri := util.SearchNFServiceUri(nfProfile, models.ServiceName_NCHF_SPENDINGLIMITCONTROL, models.NfServiceStatus_REGISTERED)
		if uri != "" {
			logger.ExtensionLog.Infof("Found CHF uri %s", uri)
			return uri
		}
	}
	return ""
}

// MakeSpendingLimitDecision will set the session rule based on chf spending decision
func MakeSpendingLimitDecision(spendingLimitStatus *models.SpendingLimitStatus, sessRule *models.SessionRule) {

	for _, policyCounterInfo := range spendingLimitStatus.StatusInfos {
		for _, customSessionRule := range factory.PcfConfig.Configuration.CustomSessionRules {
			if customSessionRule.RuleName == policyCounterInfo.CurrentStatus {
				logger.ExtensionLog.Infof("Applying custom session rule", customSessionRule.RuleName, customSessionRule.AuthSessAmbr, customSessionRule.AuthDefQos)
				sessRule.AuthSessAmbr = customSessionRule.AuthSessAmbr
				sessRule.AuthDefQos = customSessionRule.AuthDefQos
			}
		}
	}
}

// Initial Spending Limit Report to chf for policy decision and subscribe to policy counter updates
// Referenced from TS 129 513 - V15.9.0 Section 5.3.1
func InitialSpendingLimitReport(supi string, gpsi string) (*models.SpendingLimitStatus, *models.ProblemDetails) {
	// Get chf uri
	uri := getCHFUri()

	// Init CHF client
	client := util.GetNchfClient(uri)

	spendingLimitContext := models.SpendingLimitContext{
		Supi: supi,
		Gpsi: gpsi,
	}
	var response *http.Response
	spendingLimitStatus, response, err := client.DefaultApi.PostSubscription(context.Background(), spendingLimitContext)
	logger.ExtensionLog.Infof("Initial SpendingLimitStatus", spendingLimitStatus)

	if err != nil || response == nil || response.StatusCode != http.StatusCreated {
		problemDetail := util.GetProblemDetail("CHF Initial spending limit report failed", util.CHF_SUBSCRIBE_FAILED)
		logger.ExtensionLog.Warnf("CHF Initial spending limit report failed", supi)
		return nil, &problemDetail
	}
	defer func() {
		if rspCloseErr := response.Body.Close(); rspCloseErr != nil {
			logger.SMpolicylog.Errorf(
				"PostSubscription response body cannot close: %+v", rspCloseErr)
		}
	}()
	return &spendingLimitStatus, nil
}

// Intermediate Spending Limit Report to chf to retrieve updates on policy decision or modify subscription for policy counter updates
// Referenced from TS 129 513 - V15.9.0 Section 5.3.2
func IntermediateSpendingLimitReport(supi string, gpsi string) (*models.SpendingLimitStatus, *models.ProblemDetails) {
	// Get chf uri
	uri := getCHFUri()

	// Init CHF client
	client := util.GetNchfClient(uri)

	spendingLimitContext := models.SpendingLimitContext{
		Supi: supi,
		Gpsi: gpsi,
	}
	// TODO: Figure out how to retrieve subscriptionId
	subscriptionId := "some-generic-id"
	var response *http.Response
	spendingLimitStatus, response, err := client.DefaultApi.PutSubscription(context.Background(), spendingLimitContext, subscriptionId)
	logger.ExtensionLog.Infof("Intermediate SpendingLimitStatus", spendingLimitStatus)

	if err != nil || response == nil || response.StatusCode != http.StatusOK {
		problemDetail := util.GetProblemDetail("CHF Intermediate spending limit report failed", util.CHF_SUBSCRIBE_FAILED)
		logger.ExtensionLog.Warnf("CHF Intermediate spending limit report failed", supi)
		return nil, &problemDetail
	}
	defer func() {
		if rspCloseErr := response.Body.Close(); rspCloseErr != nil {
			logger.SMpolicylog.Errorf(
				"PutSubscription response body cannot close: %+v", rspCloseErr)
		}
	}()
	return &spendingLimitStatus, nil
}

// Final Spending Limit Report to chf to unsubscribe to policy counter updates
// Referenced from TS 129 513 - V15.9.0 Section 5.3.3
func FinalSpendingLimitReport(supi string, gpsi string) *models.ProblemDetails {
	// Get chf uri
	uri := getCHFUri()

	// Init CHF client
	client := util.GetNchfClient(uri)

	// TODO: Figure out how to retrieve subscriptionId
	subscriptionId := "some-generic-id"
	var response *http.Response
	response, err := client.DefaultApi.DeleteSubscription(context.Background(), subscriptionId)
	logger.ExtensionLog.Infof("FinalSpendingLimitReport response", response)

	if err != nil || response == nil || response.StatusCode != http.StatusNoContent {
		problemDetail := util.GetProblemDetail("CHF Final spending limit report failed", util.CHF_UNSUBSCRIBE_FAILED)
		logger.ExtensionLog.Warnf("CHF Final spending limit report failed", supi)
		return &problemDetail
	}
	defer func() {
		if rspCloseErr := response.Body.Close(); rspCloseErr != nil {
			logger.SMpolicylog.Errorf(
				"DeleteSubscription response body cannot close: %+v", rspCloseErr)
		}
	}()
	return nil
}
