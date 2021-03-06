package main

import (
	"flag"
	"fmt"
	"github.com/hoozecn/adslproxy"
	"net"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"
)

var once sync.Once

func PrintStackWhenSignaled() {
	once.Do(func() {
		printStackChan := make(chan os.Signal)
		signal.Notify(printStackChan, syscall.SIGUSR1)

		for {
			select {
			case <-printStackChan:
				buf := make([]byte, 1<<20)
				stackLen := runtime.Stack(buf, true)
				fmt.Printf("=== received SIGQUIT ===\n*** goroutine dump...\n%s\n*** end\n", buf[:stackLen])
			}
		}
	})
}

func main() {
	go PrintStackWhenSignaled()

	serverAddress := flag.String("server", "", "server address")
	token := flag.String("token", "", "token")
	user := flag.String("user", "demo", "username")

	//proxyUser := flag.String("proxyUser", "", "username of proxy")
	//proxyPassword := flag.String("proxyPass", "", "password of proxy")

	redialInterval := flag.Int("redialInterval", 1, "interval of redial in seconds")

	adslName := flag.String("adslName", "", "name of adsl interface (used in windows)")
	adslUsername := flag.String("adslUsername", "", "name of adsl username")
	adslPassword := flag.String("adslPassword", "", "name of adsl password")

	if *user == "" {
		*user = "demo"
	}

	flag.Set("logtostderr", "true")
	flag.Parse()

	squidLeft, _ := net.ResolveTCPAddr("tcp", "[::]:0")

	serverAddr, _ := net.ResolveTCPAddr("tcp", *serverAddress)

	client := adslproxy.NewAgent(
		*user,
		*token,
		serverAddr,
		&adslproxy.AdslConfig{
			RedialInterval: time.Duration(*redialInterval) * time.Second,
			InterfaceName:  *adslName,
			Username:       *adslUsername,
			Password:       *adslPassword,
		},
		nil,
		&adslproxy.Forward{
			Name:  "http",
			Left:  squidLeft,
			Right: "localhost:3128",
		},
	)

	for {
		client.Start()
		client.Reconnect()
	}
}
