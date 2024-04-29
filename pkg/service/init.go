package service

import (
	"context"
	"io"
	"os"
	"runtime/debug"
	"sync"

	"github.com/sirupsen/logrus"

	"github.com/free5gc/chf/internal/cgf"
	chf_context "github.com/free5gc/chf/internal/context"
	"github.com/free5gc/chf/internal/logger"
	"github.com/free5gc/chf/internal/repository"
	"github.com/free5gc/chf/internal/sbi"
	"github.com/free5gc/chf/internal/sbi/consumer"
	"github.com/free5gc/chf/internal/sbi/processor"
	"github.com/free5gc/chf/pkg/abmf"
	"github.com/free5gc/chf/pkg/rf"
)

var CHF *ChfApp

var _ App = &ChfApp{}

type App interface {
	Consumer() *consumer.Consumer
	Processor() *processor.Processor
}

type ChfApp struct {
	App
	RuntimeRepo *repository.RuntimeRepository

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	sbiServer *sbi.Server
	consumer  *consumer.Consumer
	processor *processor.Processor
}

func NewApp(ctx context.Context, RuntimeRepo *repository.RuntimeRepository, tlsKeyLogPath string) (*ChfApp, error) {
	chf := &ChfApp{
		RuntimeRepo: RuntimeRepo,
		wg:          sync.WaitGroup{},
	}
	chf.SetLogEnable(RuntimeRepo.Config().GetLogEnable())
	chf.SetLogLevel(RuntimeRepo.Config().GetLogLevel())
	chf.SetReportCaller(RuntimeRepo.Config().GetLogReportCaller())

	chf_context.Init()

	processor, err_p := processor.NewProcessor(chf, chf.RuntimeRepo)
	if err_p != nil {
		return chf, err_p
	}
	chf.processor = processor

	consumer, err := consumer.NewConsumer(chf, chf.RuntimeRepo)
	if err != nil {
		return chf, err
	}
	chf.consumer = consumer

	chf.ctx, chf.cancel = context.WithCancel(ctx)

	if chf.sbiServer, err = sbi.NewServer(chf, chf.RuntimeRepo, tlsKeyLogPath); err != nil {
		return nil, err
	}
	CHF = chf

	return chf, nil
}

func (a *ChfApp) CancelContext() context.Context {
	return a.ctx
}

func (a *ChfApp) Consumer() *consumer.Consumer {
	return a.consumer
}

func (a *ChfApp) Processor() *processor.Processor {
	return a.processor
}

func (c *ChfApp) SetLogEnable(enable bool) {
	logger.MainLog.Infof("Log enable is set to [%v]", enable)
	if enable && logger.Log.Out == os.Stderr {
		return
	} else if !enable && logger.Log.Out == io.Discard {
		return
	}

	c.RuntimeRepo.Config().SetLogEnable(enable)
	if enable {
		logger.Log.SetOutput(os.Stderr)
	} else {
		logger.Log.SetOutput(io.Discard)

	}
}

func (c *ChfApp) SetLogLevel(level string) {
	lvl, err := logrus.ParseLevel(level)
	if err != nil {
		logger.MainLog.Warnf("Log level [%s] is invalid", level)
		return
	}

	logger.MainLog.Infof("Log level is set to [%s]", level)
	if lvl == logger.Log.GetLevel() {
		return
	}

	c.RuntimeRepo.Config().SetLogLevel(level)
	logger.Log.SetLevel(lvl)
}

func (c *ChfApp) SetReportCaller(reportCaller bool) {
	logger.MainLog.Infof("Report Caller is set to [%v]", reportCaller)
	if reportCaller == logger.Log.ReportCaller {
		return
	}

	c.RuntimeRepo.Config().SetLogReportCaller(reportCaller)
	logger.Log.SetReportCaller(reportCaller)
}

func (a *ChfApp) Start() {
	logger.InitLog.Infoln("Server started")

	a.wg.Add(1)
	cgf.OpenServer(a.ctx, &a.wg)

	a.wg.Add(1)
	rf.OpenServer(a.ctx, &a.wg)

	a.wg.Add(1)
	abmf.OpenServer(a.ctx, &a.wg)

	a.wg.Add(1)
	go a.listenShutdownEvent()

	if err := a.sbiServer.Run(context.Background(), &a.wg); err != nil {
		logger.MainLog.Fatalf("Run SBI server failed: %+v", err)
	}
}

func (a *ChfApp) listenShutdownEvent() {
	defer func() {
		if p := recover(); p != nil {
			// Print stack for panic to log. Fatalf() will let program exit.
			logger.MainLog.Fatalf("panic: %v\n%s", p, string(debug.Stack()))
		}
		a.wg.Done()
	}()

	<-a.ctx.Done()
	a.Stop()
}

func (c *ChfApp) Terminate() {
	logger.MainLog.Infof("Terminating CHF...")
	// deregister with NRF
	problemDetails, err := c.Consumer().SendDeregisterNFInstance()
	if problemDetails != nil {
		logger.MainLog.Errorf("Deregister NF instance Failed Problem[%+v]", problemDetails)
	} else if err != nil {
		logger.MainLog.Errorf("Deregister NF instance Error[%+v]", err)
	} else {
		logger.MainLog.Infof("Deregister from NRF successfully")
	}
	logger.MainLog.Infof("CHF SBI Server terminated")
}

func (a *ChfApp) Stop() {
	if a.sbiServer != nil {
		a.sbiServer.Stop()
		a.Terminate()
	}
}

func (a *ChfApp) WaitRoutineStopped() {
	a.wg.Wait()
	logger.MainLog.Infof("CHF App is terminated")
}
