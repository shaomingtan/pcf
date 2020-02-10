package pcf_service

import (
	"bufio"
	"context"
	"fmt"
	"gofree5gc/lib/Nnrf_NFDiscovery"
	"gofree5gc/lib/http2_util"
	"gofree5gc/lib/openapi/common"
	"gofree5gc/lib/openapi/models"
	"gofree5gc/lib/path_util"
	"gofree5gc/src/app"
	"gofree5gc/src/pcf/AMPolicy"
	"gofree5gc/src/pcf/BDTPolicy"
	"gofree5gc/src/pcf/PolicyAuthorization"
	"gofree5gc/src/pcf/SMPolicy"
	"gofree5gc/src/pcf/UEPolicy"
	"gofree5gc/src/pcf/logger"
	"gofree5gc/src/pcf/pcf_consumer"
	"gofree5gc/src/pcf/pcf_context"
	"gofree5gc/src/pcf/pcf_handler"
	"gofree5gc/src/pcf/pcf_util"
	"os/exec"
	"sync"

	"gofree5gc/src/pcf/factory"

	"github.com/antihax/optional"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

type PCF struct{}

type (
	// Config information.
	Config struct {
		pcfcfg string
	}
)

var config Config

var pcfCLi = []cli.Flag{
	cli.StringFlag{
		Name:  "free5gccfg",
		Usage: "common config file",
	},
	cli.StringFlag{
		Name:  "pcfcfg",
		Usage: "config file",
	},
}

var initLog *logrus.Entry

func init() {
	initLog = logger.InitLog
}

func (*PCF) GetCliCmd() (flags []cli.Flag) {
	return pcfCLi
}

func (*PCF) Initialize(c *cli.Context) {

	config = Config{
		pcfcfg: c.String("pcfcfg"),
	}
	if config.pcfcfg != "" {
		factory.InitConfigFactory(path_util.Gofree5gcPath(config.pcfcfg))
	} else {
		factory.InitConfigFactory(pcf_util.PCF_CONFIG_PATH)
	}

	initLog.Traceln("PCF debug level(string):", app.ContextSelf().Logger.PCF.DebugLevel)
	if app.ContextSelf().Logger.PCF.DebugLevel != "" {
		initLog.Infoln("PCF debug level(string):", app.ContextSelf().Logger.PCF.DebugLevel)
		level, err := logrus.ParseLevel(app.ContextSelf().Logger.PCF.DebugLevel)
		if err != nil {
			logger.SetLogLevel(level)
		}
	}

	logger.SetReportCaller(app.ContextSelf().Logger.PCF.ReportCaller)
}

func (pcf *PCF) FilterCli(c *cli.Context) (args []string) {
	for _, flag := range pcf.GetCliCmd() {
		name := flag.GetName()
		value := fmt.Sprint(c.Generic(name))
		if value == "" {
			continue
		}

		args = append(args, "--"+name, value)
	}
	return args
}

func (pcf *PCF) Start() {
	initLog.Infoln("Server started")
	router := gin.Default()

	BDTPolicy.AddService(router)
	SMPolicy.AddService(router)
	AMPolicy.AddService(router)
	UEPolicy.AddService(router)
	PolicyAuthorization.AddService(router)

	self := pcf_context.PCF_Self()
	pcf_util.InitpcfContext(self)

	addr := fmt.Sprintf("%s:%d", self.HttpIPv4Address, self.HttpIpv4Port)

	profile, err := pcf_consumer.BuildNFInstance(self)
	if err != nil {
		initLog.Error("Build PCF Profile Error")
	}
	_, self.NfId, err = pcf_consumer.SendRegisterNFInstance(self.NrfUri, self.NfId, profile)
	if err != nil {
		initLog.Errorf("PCF register to NRF Error[%s]", err.Error())
	}

	// subscribe to all Amfs' status change
	amfInfos := pcf_consumer.SearchAvailableAMFs(self.NrfUri, models.ServiceName_NAMF_COMM)
	for _, amfInfo := range amfInfos {
		guamiList := pcf_util.GetNotSubscribedGuamis(amfInfo.GuamiList)
		if len(guamiList) == 0 {
			continue
		}
		client := pcf_util.GetNamfClient(amfInfo.AmfUri)
		subscriptionData := models.SubscriptionData{
			AmfStatusUri: fmt.Sprintf("%s/npcf-callback/v1/amfstatus", self.GetIPv4Uri()),
			GuamiList:    amfInfo.GuamiList,
		}
		result, httpResp, err := client.SubscriptionsCollectionDocumentApi.AMFStatusChangeSubscribe(context.Background(), subscriptionData)
		if err == nil {
			amfInfo.AmfStatusUri = result.AmfStatusUri
		} else if httpResp != nil {
			if httpResp.Status != err.Error() {
				logger.InitLog.Errorf("AMF status subscribe Error[%+v]", err)
			}
			problemDetails := err.(common.GenericOpenAPIError).Model().(models.ProblemDetails)
			logger.InitLog.Errorf("AMF status subscribe Failed[%+v]", problemDetails)
		} else {
			logger.InitLog.Errorf("%s: server no response", amfInfo.AmfUri)
		}
		self.AMFStatusSubscriptionData = append(self.AMFStatusSubscriptionData, amfInfo)
	}

	// TODO: subscribe NRF NFstatus

	go pcf_handler.Handle()
	param := Nnrf_NFDiscovery.SearchNFInstancesParamOpts{
		ServiceNames: optional.NewInterface([]models.ServiceName{models.ServiceName_NUDR_DR}),
	}
	resp, err := pcf_consumer.SendSearchNFInstances(self.NrfUri, models.NfType_UDR, models.NfType_PCF, param)
	for _, nfProfile := range resp.NfInstances {
		udruri := pcf_util.SearchNFServiceUri(nfProfile, models.ServiceName_NUDR_DR, models.NfServiceStatus_REGISTERED)
		if udruri != "" {
			self.DefaultUdrUri = udruri
			break
		}
	}
	if err != nil {
		initLog.Errorln(err)
	}
	server, err := http2_util.NewServer(addr, pcf_util.PCF_LOG_PATH, router)
	if err == nil && server != nil {
		initLog.Infoln(server.ListenAndServeTLS(pcf_util.PCF_PEM_PATH, pcf_util.PCF_KEY_PATH))
	}
	if err != nil {
		initLog.Errorln(err)
	}
}

func (pcf *PCF) Exec(c *cli.Context) error {
	initLog.Traceln("args:", c.String("pcfcfg"))
	args := pcf.FilterCli(c)
	initLog.Traceln("filter: ", args)
	command := exec.Command("./pcf", args...)

	stdout, err := command.StdoutPipe()
	if err != nil {
		initLog.Fatalln(err)
	}
	wg := sync.WaitGroup{}
	wg.Add(4)
	go func() {
		in := bufio.NewScanner(stdout)
		for in.Scan() {
			fmt.Println(in.Text())
		}
		wg.Done()
	}()

	stderr, err := command.StderrPipe()
	if err != nil {
		initLog.Fatalln(err)
	}
	go func() {
		in := bufio.NewScanner(stderr)
		fmt.Println("PCF log start")
		for in.Scan() {
			fmt.Println(in.Text())
		}
		wg.Done()
	}()

	go func() {
		fmt.Println("PCF start")
		if err := command.Start(); err != nil {
			fmt.Printf("command.Start() error: %v", err)
		}
		fmt.Println("PCF end")
		wg.Done()
	}()

	wg.Wait()

	return err
}
