package appService

import (
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"time"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/debug"
	"golang.org/x/sys/windows/svc/eventlog"
	"golang.org/x/sys/windows/svc/mgr"
)

type Handler interface {
	Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errNo uint32)
}

type ServiceConfig struct {
	name        string
	displayName string
	description string
}

func NewConfig(name, displayName, description string) ServiceConfig {
	return ServiceConfig{
		name:        name,
		displayName: displayName,
		description: description,
	}
}

type AppService struct {
	config  ServiceConfig
	handler Handler
}

func NewService(config ServiceConfig, hadler Handler) *AppService {
	return &AppService{config: config, handler: hadler}
}

func (appSrv *AppService) RunService(isDebug bool) error {
	var err error
	var elog debug.Log

	if isDebug {
		elog = debug.New(appSrv.config.name)
	} else {
		elog, err = eventlog.Open(appSrv.config.name)
		if err != nil {
			return err
		}
	}
	defer elog.Close()

	msg := fmt.Sprintf("starting %s service", appSrv.config.name)
	elog.Info(1, msg)
	slog.Info(msg)

	run := svc.Run
	if isDebug {
		run = debug.Run
	}
	err = run(appSrv.config.name, appSrv.handler)
	if err != nil {
		msg = fmt.Sprintf("%s service failed: %v", appSrv.config.name, err)
		elog.Error(1, msg)
		slog.Error(msg)
		return err
	}
	msg = fmt.Sprintf("%s service stopped", appSrv.config.name)
	elog.Info(1, msg)
	slog.Info(msg)

	return nil
}

func (appSvc *AppService) IsService() (bool, error) {
	os := runtime.GOOS
	switch os {
	case "windows":
		return svc.IsWindowsService()
	case "linux":
		return false, fmt.Errorf("")
	case "darwin":
		return false, nil
	case "freebsd", "openbsd", "netbsd":
		return false, nil
	case "android":
		return false, nil
	case "ios":
		return false, nil
	default:
		return false, nil
	}
}

func (appSrv *AppService) Install() error {
	exePath, err := os.Executable()
	if err != nil {
		return err
	}

	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()

	service, err := m.OpenService(appSrv.config.name)
	if err == nil {
		service.Close()
		return fmt.Errorf("service %s already exists", appSrv.config.name)
	}

	service, err = m.CreateService(
		appSrv.config.name,
		exePath,
		mgr.Config{
			DisplayName: appSrv.config.displayName,
			Description: appSrv.config.description,
			StartType:   mgr.StartAutomatic,
		},
	)
	if err != nil {
		return err
	}
	defer service.Close()

	recoveryActions := []mgr.RecoveryAction{
		{
			Type:  mgr.ServiceRestart,
			Delay: 1 * time.Minute,
		},
		{
			Type:  mgr.ServiceRestart,
			Delay: 1 * time.Minute,
		},
		{
			Type:  mgr.ServiceRestart,
			Delay: 1 * time.Minute,
		},
	}
	err = service.SetRecoveryActions(recoveryActions, 0)
	if err != nil {
		return fmt.Errorf("failed to configure recovery policy: %v", err)
	}

	err = eventlog.InstallAsEventCreate(appSrv.config.name, eventlog.Error|eventlog.Warning|eventlog.Info)
	if err != nil {
		service.Delete()
		return fmt.Errorf("SetupEventLogSource() failed: %s", err)
	}

	return nil
}

func (appSrv *AppService) Uninstall() error {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()

	service, err := m.OpenService(appSrv.config.name)
	if err != nil {
		return fmt.Errorf("service %s is not installed", appSrv.config.name)
	}
	defer service.Close()

	err = service.Delete()
	if err != nil {
		return err
	}

	err = eventlog.Remove(appSrv.config.name)
	if err != nil {
		return fmt.Errorf("RemoveEventLogSource() failed: %s", err)
	}

	return nil
}
