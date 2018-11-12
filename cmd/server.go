package main

import (
	"flag"
	"fmt"
	"github.com/hoozecn/adslproxy"
	"net"
)

func main() {
	sshPort := flag.Int("sshPort", 11222, "ssh port")
	httpPort := flag.Int("httpPort", 11280, "ssh port")
	token := flag.String("token", "", "ssh port")

	flag.Set("logtostderr", "true")
	flag.Parse()

	sshAddr, _ := net.ResolveTCPAddr("tcp", fmt.Sprintf("[::]:%d", *sshPort))
	httpAddr, _ := net.ResolveTCPAddr("tcp", fmt.Sprintf("[::]:%d", *httpPort))
	s := adslproxy.NewServer(sshAddr, httpAddr, *token)
	s.Start()
}
