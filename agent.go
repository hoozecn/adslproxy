package adslproxy

import (
	"github.com/gocloudio/crypto/ssh"
	"github.com/golang/glog"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"io"
	"net"
	"sync"
)

type Agent struct {
	sshConfig   *ssh.ClientConfig
	ForwardList []*Forward
	serverAddr  *net.TCPAddr
	stopper     chan bool

	id   string
	user string
}

func NewForward(name, left, right string) (*Forward, error) {
	leftAddr, err := net.ResolveTCPAddr("tcp", left)
	return &Forward{
		Left:  leftAddr,
		Right: right,
		Name:  name,
	}, err
}

func NewAgent(user, password string, serverAddr *net.TCPAddr, forwards ... *Forward) *Agent {
	var config *ssh.ClientConfig
	id := uuid.New().String()

	config = &ssh.ClientConfig{
		User:            user + "@" + id,
		Auth:            []ssh.AuthMethod{ssh.Password(password),},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         ClientConnectTimeout,
	}

	return &Agent{
		sshConfig:   config,
		ForwardList: forwards,
		serverAddr:  serverAddr,
	}
}

func (a *Agent) Stop() {
	close(a.stopper)
}

func (a *Agent) Start() error {
	a.stopper = make(chan bool)

	client, err := ssh.Dial("tcp", a.serverAddr.String(), a.sshConfig)
	if err != nil {
		glog.Errorf("failed to connect to %s %s", a.serverAddr.String(), err)
		return errors.WithStack(err)
	}

	glog.Infof("connected to %s", a.serverAddr.String())

	errc := make(chan bool, len(a.ForwardList))
	defer close(errc)

	go func() {
		select {
		case <-a.stopper:
			client.Close()
		case <-errc:
			client.Close()
		}
	}()

	wg := sync.WaitGroup{}
	wg.Add(len(a.ForwardList))

	for _, forward := range a.ForwardList {
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
				client, err := l.Accept()
				if err != nil {
					errc <- true
					return
				}

				go func() {
					defer client.Close()

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

					go pipe(client, local)
					go pipe(local, client)

					<-chDone
				}()
			}
		}(forward, listener)
	}

	wg.Wait()
	client.Wait()
	return nil
}
