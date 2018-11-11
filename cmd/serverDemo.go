package main

import (
	"flag"
	"github.com/hoozecn/adslproxy"
	"net"
	"time"
)

func main() {
	flag.Parse()
	addr, _ := net.ResolveTCPAddr("tcp", "0.0.0.0:8899")
	ticker := time.NewTicker(10 * time.Second)
	s := adslproxy.NewServer(addr, "123456")

	go func() {
		for {
			select {
			case <-ticker.C:
				s.ClearNodes()
			}
		}
	}()

	s.Start()
}
