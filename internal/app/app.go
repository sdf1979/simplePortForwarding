package app

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	_cfg "github.com/sdf1979/config"
)

const version = "1.0.2"

type Config struct {
	LocalPort   int `json:"localPort"`
	LifeTimeId  int `json:"lifeTimeId"`
	RemoteHosts []struct {
		ID   string `json:"id"`
		Host string `json:"host"`
		Port int    `json:"port"`
	} `json:"remoteHosts"`
	currentId string
	mu        sync.RWMutex
}

func (config *Config) SetCurrentId(id string) {
	config.mu.Lock()
	defer config.mu.Unlock()
	config.currentId = id
}

func (config *Config) ClearCurrentId(seconds int, id string) {
	time.Sleep(time.Duration(seconds) * time.Second)
	config.mu.Lock()
	defer config.mu.Unlock()
	if config.currentId == id {
		config.currentId = ""
		slog.Info("ID value cleared")
	}
}

func (config *Config) GetCurrentId() string {
	config.mu.RLock()
	defer config.mu.RUnlock()
	return config.currentId
}

func (config *Config) FindRemoteHostByID(id string) (host string, port int, err error) {
	config.mu.RLock()
	defer config.mu.RUnlock()

	for _, host := range config.RemoteHosts {
		if host.ID == id {
			return host.Host, host.Port, nil
		}
	}

	return "", -1, errors.New("remote host not found with ID: " + id)
}

var config *Config

func Run(ctx context.Context) {
	var err error
	config, err = _cfg.LoadConfig[Config]()
	if err != nil {
		slog.Error("failed to load config: " + err.Error())
		os.Exit(1)
	}

	startForwarding(ctx, config)

	slog.Info(fmt.Sprintf("server ver. %s stopped", version))

}

func startForwarding(ctx context.Context, config *Config) {
	lc := &net.ListenConfig{}
	listener, err := lc.Listen(ctx, "tcp", fmt.Sprintf(":%d", config.LocalPort))
	if err != nil {
		slog.Error(fmt.Sprintf("Error starting server on localhost:%d: %v", config.LocalPort, err))
		return
	}
	defer listener.Close()
	slog.Info(fmt.Sprintf("Port forwarding is running on localhost:%d", config.LocalPort))

	go func() {
		<-ctx.Done()
		listener.Close()
	}()

	for {
		clientConn, err := listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				slog.Info("Port forwarding stopped")
				return
			}
			slog.Error(fmt.Sprintf("Error accepting connection: %v", err))
			continue
		}

		reader := bufio.NewReader(clientConn)
		isHTTP, err := isHTTPRequest(reader)
		if err != nil {
			slog.Error(fmt.Sprintf("Connection type check error: %v", err))
			clientConn.Close()
			continue
		}
		if isHTTP {
			id := handleConnectionHttp(reader, clientConn)
			config.SetCurrentId(id)
			if config.LifeTimeId != 0 {
				go config.ClearCurrentId(config.LifeTimeId, id)
			}
		} else {
			host, port, err := config.FindRemoteHostByID(config.GetCurrentId())
			if err != nil {
				slog.Error(fmt.Sprintf("%v", err))
				continue
			}
			go handleConnection(reader, clientConn, host, strconv.Itoa(port))
		}
	}
}

func isHTTPRequest(reader *bufio.Reader) (bool, error) {
	data, err := reader.Peek(16)
	if err != nil {
		if err == io.EOF {
			return false, nil
		}
		// Для timeout считаем, что это не HTTP
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			return false, nil
		}
		return false, err
	}

	// Проверяем HTTP методы
	methods := []string{"GET ", "POST", "PUT ", "HEAD", "OPTI"}
	firstBytes := string(data)

	for _, method := range methods {
		if strings.HasPrefix(firstBytes, method) {
			return true, nil
		}
	}

	// Дополнительная проверка: если есть не-ASCII символы, это не HTTP
	for i := 0; i < len(data) && i < 8; i++ {
		if data[i] < 32 || data[i] > 126 {
			// Найден не-печатный символ - вероятно бинарный протокол
			return false, nil
		}
		if data[i] == 0 {
			// Нулевой байт - точно не HTTP
			return false, nil
		}
	}

	return false, nil
}

func handleConnectionHttp(reader *bufio.Reader, conn net.Conn) string {
	defer conn.Close()

	req, err := http.ReadRequest(reader)
	if err != nil {
		slog.Error(fmt.Sprintf("Error reading HTTP request: %v", err))
		return ""
	}

	id := req.URL.Query().Get("id")
	slog.Info(fmt.Sprintf("Received HTTP request with id: %s", id))
	config.SetCurrentId(id)

	response := "HTTP/1.1 200 OK\r\nContent-Length: 0\r\n\r\n"
	conn.Write([]byte(response))
	return id
}

func handleConnection(reader *bufio.Reader, clientConn net.Conn, remoteHost, remotePort string) {
	defer clientConn.Close()

	remoteAddr := net.JoinHostPort(remoteHost, remotePort)

	remoteConn, err := net.Dial("tcp", remoteAddr)
	if err != nil {
		slog.Error(fmt.Sprintf("Error connecting to %s: %v", remoteAddr, err))
		return
	}
	defer remoteConn.Close()

	slog.Info(fmt.Sprintf("Connection: %s -> %s", clientConn.RemoteAddr(), remoteAddr))

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()

		if reader.Buffered() > 0 {
			buffered := make([]byte, reader.Buffered())
			n, err := reader.Read(buffered)
			if err == nil && n > 0 {
				remoteConn.Write(buffered[:n])
			}
		}

		io.Copy(remoteConn, clientConn)
		remoteConn.(*net.TCPConn).CloseWrite()
	}()

	go func() {
		defer wg.Done()
		io.Copy(clientConn, remoteConn)
		clientConn.(*net.TCPConn).CloseWrite()
	}()

	wg.Wait()
	slog.Info(fmt.Sprintf("Disconnection: %s", clientConn.RemoteAddr()))
}

func GetVersion() string {
	return version
}
