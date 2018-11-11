package main

import (
	"flag"
	"github.com/hoozecn/adslproxy"
	"net"
	"time"
)

func main() {
	flag.Parse()

	serverAddr, _ := net.ResolveTCPAddr("tcp", "127.0.0.1:8899")
	forward1, _ := adslproxy.NewForward("http", "[::]:0", "127.0.0.1:8888")
	forward2, _ := adslproxy.NewForward("socks5", "[::]:0", "127.0.0.1:8889")
	client := adslproxy.NewAgent("demo", "123456", serverAddr, forward1, forward2)

	for {
		client.Start()
		time.Sleep(5 * time.Second)
	}
}
