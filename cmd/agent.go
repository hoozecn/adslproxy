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
	user := flag.String("user", "", "username")

	proxyUser := flag.String("proxyUser", "", "username of proxy")
	proxyPassword := flag.String("proxyUser", "", "password of proxy")

	flag.Set("logtostderr", "true")
	flag.Parse()

	serverAddr, _ := net.ResolveTCPAddr("tcp", *serverAddress)

	if *user == "" {
		*user = "demo"
	}

	client := adslproxy.NewAgent(
		*user,
		*token,
		serverAddr,
		&adslproxy.AdslConfig{
			RedialInterval: 1 * time.Second,
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
