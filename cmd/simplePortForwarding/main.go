package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"simple_port_forwarding/internal/app"
	"syscall"
	"time"

	"github.com/sdf1979/appService"
	_log "github.com/sdf1979/logger"
	"golang.org/x/sys/windows/svc"
)

type Args struct {
	Install     bool
	Uninstall   bool
	Version     bool
	Help        bool
	Name        string
	DisplayName string
	Description string
}

func (args *Args) Parse() {
	flag.BoolVar(&args.Install, "install", false, "Install the service")
	flag.BoolVar(&args.Uninstall, "uninstall", false, "Uninstall the service")
	flag.BoolVar(&args.Version, "version", false, "version the service")
	flag.BoolVar(&args.Help, "help", false, "Show help message")
	flag.StringVar(&args.Name, "name", "Simple Port Forwarding", "Service name")
	flag.StringVar(&args.DisplayName, "display_name", "", "Service display name (default value of argument \"-name\")")
	flag.Parse()
	if args.DisplayName == "" {
		args.DisplayName = args.Name
	}
	args.Description = "Simple port forwarding"
}

func (args *Args) printHelp() {
	fmt.Println("Options:")
	fmt.Println("  -install       Install the service")
	fmt.Println("  -uninstall     Uninstall the service")
	fmt.Println("  -version       Show version information")
	fmt.Println("  -help          Show this help message")
	fmt.Println("  -name string   Service name (default: \"Simple Port Forwarding\")")
	fmt.Println("  -display_name string Service display name")
}

type serviceOperation func() error

func handleServiceOperation(operation serviceOperation, name, successMsg, errorMsg string) {
	err := operation()
	if err != nil {
		msg := fmt.Sprintf(errorMsg, err)
		fmt.Println(msg)
		slog.Error(msg)
		return
	}
	msg := fmt.Sprintf(successMsg, name)
	fmt.Println(msg)
	slog.Info(msg)
}

type MyHandler struct{}

func (handler *MyHandler) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown | svc.AcceptPauseAndContinue
	changes <- svc.Status{State: svc.StartPending}
	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go app.Run(ctx)

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
				cancel()
				break loop
			case svc.Pause:
				cancel()
				changes <- svc.Status{State: svc.Paused, Accepts: cmdsAccepted}
			case svc.Continue:
				ctx, cancel = context.WithCancel(context.Background())
				go app.Run(ctx)
				changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
			default:
				slog.Error("unexpected control request", "command", c)
			}
		default:
			time.Sleep(time.Millisecond * 100)
		}
	}
	cancel()
	return
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-shutdown
		cancel()
	}()

	_log.InitLogger(ctx, "logs")

	args := Args{}
	args.Parse()

	if args.Help {
		args.printHelp()
		return
	}

	svcConfig := appService.NewConfig(args.Name, args.DisplayName, args.Description)
	service := appService.NewService(svcConfig, &MyHandler{})

	if args.Install {
		handleServiceOperation(service.Install, args.Name, "service %s installed", "failed to install command: %v")
		return
	} else if args.Uninstall {
		handleServiceOperation(service.Uninstall, args.Name, "service %s uninstalled", "failed to uninstall command: %v")
		return
	} else if args.Version {
		fmt.Printf("v%s\n", app.GetVersion())
		return
	}

	isService, err := service.IsService()
	if err != nil {
		slog.Error(fmt.Sprintf("failed to determine if we are running as a service: %v", err))
		return
	}

	if isService {
		service.RunService(false)
		return
	}

	app.Run(ctx)
}
