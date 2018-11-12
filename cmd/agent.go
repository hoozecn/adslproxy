package main

import (
	"flag"
	"github.com/hoozecn/adslproxy"
	"net"
	"time"
)

func main() {
	serverAddress := flag.String("server", "", "server address")
	token := flag.String("token", "", "token")
	user := flag.String("user", "demo", "username")

	proxyUser := flag.String("proxyUser", "", "username of proxy")
	proxyPassword := flag.String("proxyPass", "", "password of proxy")

	redialInterval := flag.Int("redialInterval", 1, "interval of redial in seconds")

	adslName := flag.String("adslName", "", "name of adsl interface (used in windows)")
	adslUsername := flag.String("adslUsername", "", "name of adsl username")
	adslPassword := flag.String("adslPassword", "", "name of adsl password")

	if *user == "" {
		*user = "demo"
	}

	flag.Set("logtostderr", "true")
	flag.Parse()

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
		&adslproxy.ProxyCredential{
			Username: *proxyUser,
			Password: *proxyPassword,
		},
	)

	for {
		client.Start()
		client.Reconnect()
	}
}
