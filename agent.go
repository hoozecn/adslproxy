package adslproxy

import (
	"fmt"
	"github.com/gocloudio/crypto/ssh"
	"github.com/golang/glog"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"io"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/armon/go-socks5"
	"github.com/elazarl/goproxy"
	goproxyAuth "github.com/elazarl/goproxy/ext/auth"
)

type Agent struct {
	sshConfig   *ssh.ClientConfig
	ForwardList []*Forward
	serverAddr  *net.TCPAddr
	stopper     chan bool

	id         string
	user       string
	adslConfig *AdslConfig

	connectMutex sync.Mutex

	httpProxyListener  net.Listener
	socksProxyListener net.Listener
	proxyCredential    *ProxyCredential
}

func NewForward(name, left, right string) (*Forward, error) {
	leftAddr, err := net.ResolveTCPAddr("tcp", left)
	return &Forward{
		Left:  leftAddr,
		Right: right,
		Name:  name,
	}, err
}

func NewAgent(
	user, password string,
	serverAddr *net.TCPAddr,
	adslConfig *AdslConfig,
	credential *ProxyCredential,
	forwards ... *Forward,
) *Agent {
	var config *ssh.ClientConfig
	id := uuid.New().String()

	config = &ssh.ClientConfig{
		User:            user + "@" + id,
		Auth:            []ssh.AuthMethod{ssh.Password(password),},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         ClientConnectTimeout,
	}

	return &Agent{
		sshConfig:       config,
		ForwardList:     forwards,
		serverAddr:      serverAddr,
		adslConfig:      adslConfig,
		proxyCredential: credential,
	}
}

func (a *Agent) Stop() {
	close(a.stopper)
}

func (a *Agent) Dial() (*ssh.Client, error) {
	addr := a.serverAddr.String()
	conn, err := net.DialTimeout("tcp", addr, a.sshConfig.Timeout)
	if err != nil {
		return nil, err
	}

	c, chans, reqs, err := ssh.NewClientConn(
		&Conn{
			Conn:         conn,
			ReadTimeout:  ReadTimeout,
			WriteTimeout: WriteTimeout,
		},
		addr,
		a.sshConfig,
	)

	if err != nil {
		return nil, err
	}

	return ssh.NewCustomReqHandlerClient(c, chans, reqs, func(reqs <-chan *ssh.Request) {
		a.SshRequestHandler(c, reqs)
	}), nil
}

func (a *Agent) SshRequestHandler(c ssh.Conn, reqs <-chan *ssh.Request) {
	for req := range reqs {
		switch req.Type {
		case Reconnect:
			req.Reply(true, nil)
			c.Close()
			return
		default:
			req.Reply(true, nil)
		}
	}
}

func (a *Agent) Start() error {
	a.stopper = make(chan bool)

	forwardList := a.ForwardList[:]

	if a.proxyCredential != nil {
		addr := a.StartHttpProxy()
		f, _ := NewForward("http", "[::]:0", fmt.Sprintf("localhost:%d", addr.Port), )
		forwardList = append(forwardList, f)
		defer a.StopHttpProxy()
	}

	if a.proxyCredential != nil {
		addr := a.StartSocksProxy()
		f, _ := NewForward("socks5", "[::]:0", fmt.Sprintf("localhost:%d", addr.Port), )
		forwardList = append(forwardList, f)
		defer a.StopSocks5Proxy()
	}

	client, err := a.Dial()

	if err != nil {
		glog.Errorf("failed to connect to %s %s", a.serverAddr.String(), err)
		return errors.WithStack(err)
	}

	glog.Infof("connected to %s", a.serverAddr.String())

	errc := make(chan bool, len(forwardList))
	defer close(errc)

	go func() {
		select {
		case <-a.stopper:
			client.Close()
		case <-errc:
			client.Close()
		}
	}()

	ticker := time.NewTicker(HeartbeatInterval)
	defer ticker.Stop()

	go func() {
		for {
			select {
			case <-ticker.C:
				ret, _, err := client.SendRequest("keepalive", true, nil)
				if !ret || err != nil {
					client.Close()
					ticker.Stop()
					return
				} else {
					glog.V(2).Infof("keep alive %s", time.Now())
				}
			}
		}
	}()

	wg := sync.WaitGroup{}
	wg.Add(len(forwardList))

	for _, forward := range forwardList {
		listener, err := client.CreateTunnel(forward.Left, forward.Right, forward.Name)
		if err != nil {
			glog.Errorf("failed to listen to server port %s %s", forward.Right, err)
			return err
		}

		// fix when port is 0
		forward.Left = listener.Addr().(*net.TCPAddr)

		go func(t *Forward, l net.Listener) {
			defer func() {
				wg.Done()
				l.Close()
				glog.Infof("tunnel is closed %s", t)
			}()

			glog.Infof("new tunnel created %s", t)

			// handle incoming connections on reverse forwarded tunnel
			for {
				remote, err := l.Accept()
				if err != nil {
					errc <- true
					return
				}

				go func() {
					defer remote.Close()

					// Open a (local) connection to localEndpoint whose content will be forwarded so serverEndpoint
					local, err := net.Dial("tcp", t.Right)
					if err != nil {
						glog.Errorf("failed to connect to service %s %s", t.Right, err)
						return
					}

					defer local.Close()

					chDone := make(chan bool, 1)

					pipe := func(to io.WriteCloser, from io.ReadCloser) {
						// if either end breaks, close both ends to ensure they're both unblocked,
						// otherwise io.Copy can block forever if e.g. reading after write end has
						// gone away
						defer func() {
							chDone <- true
						}()

						_, _ = io.Copy(to, from)
					}

					go pipe(remote, local)
					go pipe(local, remote)

					<-chDone
				}()
			}
		}(forward, listener)
	}

	wg.Wait()
	client.Wait()
	return nil
}

func (a *Agent) Reconnect() {
	if a.adslConfig != nil {
		for {
			err := a.adslConfig.Redial()
			if err != nil {
				glog.Errorf("failed to redial %s", err)
				time.Sleep(a.adslConfig.RedialInterval)
				continue
			}

			break
		}
	}
}
func (a *Agent) StartHttpProxy() *net.TCPAddr {
	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		panic(err)
	}

	go func() {
		proxy := goproxy.NewProxyHttpServer()

		if a.proxyCredential.Username != "" {
			goproxyAuth.ProxyBasic(proxy, "Adslproxy", func(user, passwd string) bool {
				if user == a.proxyCredential.Username && passwd == a.proxyCredential.Password {
					return true
				}

				return false
			})
		}

		http.Serve(l, proxy)
	}()

	a.httpProxyListener = l

	return l.Addr().(*net.TCPAddr)
}
func (a *Agent) StartSocksProxy() *net.TCPAddr {
	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		panic(err)
	}

	config := &socks5.Config{}
	if a.proxyCredential.Username != "" {
		config.AuthMethods = []socks5.Authenticator{
			socks5.UserPassAuthenticator{
				Credentials: socks5.StaticCredentials{a.proxyCredential.Username: a.proxyCredential.Password},
			},
		}
	}

	server, _ := socks5.New(config)

	go func() {
		server.Serve(l)
	}()

	a.socksProxyListener = l
	return l.Addr().(*net.TCPAddr)
}

func (a *Agent) StopHttpProxy() {
	a.httpProxyListener.Close()
}

func (a *Agent) StopSocks5Proxy() {
	a.socksProxyListener.Close()
}

type ProxyCredential struct {
	Username string
	Password string
}
