package appService

import (
	"log/slog"
	"time"

	"golang.org/x/sys/windows/svc"
)

type defaultHandler struct{}

func GetDefaultHandler() *defaultHandler {
	return &defaultHandler{}
}

func (handler *defaultHandler) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown | svc.AcceptPauseAndContinue
	changes <- svc.Status{State: svc.StartPending}
	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
loop:
	for {
		select {
		case c, ok := <-r:
			if !ok {
				break loop
			}
			switch c.Cmd {
			case svc.Interrogate:
				changes <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				break loop
			case svc.Pause:
				changes <- svc.Status{State: svc.Paused, Accepts: cmdsAccepted}
			case svc.Continue:
				changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
			default:
				slog.Error("unexpected control request", "command", c)
			}
		default:
			time.Sleep(time.Millisecond * 100)
		}
	}
	return
}
