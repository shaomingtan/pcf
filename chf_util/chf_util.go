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

// Subscribe to chf SpendingLimit notifications
func SubscribeSpendingLimit(supi string, gpsi string) (*models.SpendingLimitStatus, *models.ProblemDetails) {
	// Get chf uri
	uri := getCHFUri()

	// Init CHF client
	client := util.GetNchfClient(uri)

	spendingLimitContext := models.SpendingLimitContext{
		Supi: supi,
		Gpsi: gpsi,
	}
	var response *http.Response
	spendingLimitStatus, response, err := client.DefaultApi.Subscribe(context.Background(), spendingLimitContext)
	logger.ExtensionLog.Infof("SpendingLimitStatus", spendingLimitStatus)

	if err != nil || response == nil || response.StatusCode != http.StatusCreated {
		problemDetail := util.GetProblemDetail("CHF spending limit control subscribe failed", util.CHF_SUBSCRIBE_FAILED)
		logger.ExtensionLog.Warnf("CHF spending limit control subscribe failed", supi)
		return nil, &problemDetail
	}
	defer func() {
		if rspCloseErr := response.Body.Close(); rspCloseErr != nil {
			logger.SMpolicylog.Errorf(
				"SpendingLimitSubscribe response body cannot close: %+v", rspCloseErr)
		}
	}()
	return &spendingLimitStatus, nil
}
