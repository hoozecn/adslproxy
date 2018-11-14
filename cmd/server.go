package main

import (
	"flag"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/hoozecn/adslproxy"
	"net"
	"net/http"
)

func ServePac(addr *net.TCPAddr) {
	l, err := net.ListenTCP("tcp", addr)

	if err != nil {
		panic(err)
	}

	r := mux.NewRouter()

	r.HandleFunc("/pac/{proxy}/", func(writer http.ResponseWriter, request *http.Request) {
		vars := mux.Vars(request)
		proxy := vars["proxy"]

		writer.Header().Set("Content-Type", "text/plain")
		writer.WriteHeader(200)
		writer.Write([]byte(fmt.Sprintf(`function FindProxyForURL(url, host) {
	if (shExpMatch(host, "*cdn*") || shExpMatch(host, "*.sinaimg.cn") || shExpMatch(host, "*.sinajs.cn") || shExpMatch(host, "*.cmvideo.cn")) {
		return "DIRECT";
	}
	return "PROXY %s;";
}`, proxy)))
	})

	http.Serve(l, r)
}

func main() {
	sshPort := flag.Int("sshPort", 11222, "ssh port")
	httpPort := flag.Int("httpPort", 11280, "http port")
	pacPort := flag.Int("pacPort", 11281, "pac file port")
	token := flag.String("token", "", "ssh token")

	flag.Set("logtostderr", "true")
	flag.Parse()

	sshAddr, _ := net.ResolveTCPAddr("tcp", fmt.Sprintf("[::]:%d", *sshPort))
	httpAddr, _ := net.ResolveTCPAddr("tcp", fmt.Sprintf("[::]:%d", *httpPort))
	pacAddr, _ := net.ResolveTCPAddr("tcp", fmt.Sprintf("[::]:%d", *pacPort))

	go func() {
		ServePac(pacAddr)
	}()

	s := adslproxy.NewServer(sshAddr, httpAddr, *token)
	s.Start()
}
