package adslproxy

import (
	"container/list"
	"fmt"
	"github.com/gocloudio/crypto/ssh"
	"github.com/golang/glog"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Node represents an agent registered in server
type Node struct {
	// id of the agent
	Id string
	// Name of the agent
	Name        string
	RemoteIp    string
	ForwardList []*Forward
	// Heartbeat is the time of last heartbeat
	Heartbeat time.Time

	conn   *ssh.ServerConn
	ticker *time.Ticker
}

func (n *Node) Format(s fmt.State, c rune) {
	fmt.Fprintf(s, "%s@%s[%s]", n.Name, n.RemoteIp, n.Id)
}

func (n *Node) IsAlive() bool {
	return n.Heartbeat.After(time.Now().Add(-HeartbeatInterval))
}

func (n *Node) Clear() {
	n.ticker.Stop()

	for _, forward := range n.ForwardList {
		forward.listener.Close()
	}

	n.conn.Close()
}

func (n *Node) AddForwarding(msg ssh.NamedTunnelForwardMsg, listener *net.TCPListener) {
	f := &Forward{
		Name:     msg.Name,
		Left:     listener.Addr().(*net.TCPAddr),
		Right:    msg.Right,
		Options:  msg.Options,
		listener: listener,
	}

	n.ForwardList = append(n.ForwardList, f)
	glog.Infof("A new forwarding is added %s via %s", f, n)
}

func (n *Node) Redial() {
	n.conn.SendRequest(Reconnect, true, nil)
	n.conn.Close()
}

type Server struct {
	// SshAddr is the addr that agent registered to
	SshAddr *net.TCPAddr
	Nodes   *list.List

	// HttpAddr is the addr of the api
	HttpAddr *net.TCPAddr

	sshConfig    *ssh.ServerConfig
	stopped      bool
	sshListener  *net.TCPListener
	nodeOps      sync.Mutex
	httpListener *net.TCPListener
}

const hostPrivateKey = `-----BEGIN RSA PRIVATE KEY-----
Proc-Type: 4,ENCRYPTED
DEK-Info: AES-128-CBC,F841237AC0C79E2753411313BB70DA4E

CJw6+cySysh4edy7HPcCPsrg4jqmWuMgy9QbAZJVLpcBFUBQOobJnYrjzYV0p3bL
3JBoZRIO5eowCyQSeTk5oeNkKVCGboU9kG5pdT9fBK6GCaSO1++kZ6FwlJ3+6CUO
JblEHwe3da6gDwgt3MhWxBXdCY2Y+o6isrCo/UeSlo4yfTEC93KmoVIUg89/GUAC
4uhmx2FpBC2XyPZu71Ut/drJzHXoVlPMTW1nd2yybUVrjcsS1AI4DQ4IK4BrX0Y3
tm22PAQ2WX6CGJT3wbReyhupISV1JSjEST6JasaSHPuKd15gC85FG8zeU3ASbye1
+LejQAIOl4bjy3Uph7p3KjdGCRaFjrPNyUdaGV2gxFmgwpnCA2bX6H980TKD+Pti
wP1+GAl3iZHocoAkxaZAkvC+GyXsKqYq3h/I3cYmwv/nnYnV2SS/CLnSfsypQVNi
fc3aVD/qcQMwAzc2W5MIRyR0lqGURem6Y0DEP0irQQrlVkjk8TAAtXKmtSul7UFQ
rnul5YkE3UkLrfHvSXHyt8utJUclm2gFGAgn8iS1l7WRBv1pWbbdZvoyp7Alhe7M
7C8+uRr6niDAPZqbUrsa4zrHnikb0qkSzK8VrbkWwXl24E1Z1a9SMlH6V0zrgPbJ
lSeAYLevmBTBswesXWy3BwGonJudCqrM1vpjU1NAb9hyheozbpdYxr3/PqUPz8sd
0GVeOsHn7CtC03x8WlX8+YC6Sur3/Rp8EgVZEYi+rsjSB37J/MiI47jjo1MfOMWk
Ud0GixZ3WjxRmxV8lYG3peOpu1HqvwtRG5TP2t9h4Q50BPvwDVi65tdNLufDF1iT
NgGgpqW1xq5I0FSjP3BTWZoGsDi4vGm9oC73rcF+gAN/35sNWkNXaFPig6zM5o+s
O9tPFx5BIoll8kVQrwIbFoD2o1xpa0HODQHB3tiG7CUBaC1Npgblfi1ppSzjkXKs
Pjxs2R7goKoFujEAHqkdPVH0GDY5ZTZ//uo2+pJ4/0+eTA8NVjKaxOeGkAs1SS/u
dzretwOcsH0pu+M06pZqFZa+CVP8c9h4ENtys1Yh6dCoUuVqQ1nFV/pPteit6rU8
giVJO9erY9tqMO2nj4DbxZAun0F1zNrZrH+fP6CdprsqvnlkYQjhU3jq5ds60ozg
wbuGjYGkOv42mGQfjpFzilu+970kiopQ6TgbRESmvBr4LT7uJVxIl9wBeums7BbX
TlCNMugzJfxq55N5p4CnYjGoLoWch8mPUiW96JNPJ3qYFA5JMtaJbDEHZ8hrYhnq
vtbo3vMwI9sBDSNfiH5QiWKsf/1GaRI6Wm5b3vLvZhiTLhGQDJTPPTb08e6M/vPF
LYEy7H8JFI+tgjIxbEq0ACo297ltw6kq+g2KJOs0ANsPmGSkXAq/X4/I4Bb+cNoV
WrTV9W4PHKyxsnPTZ+o0RTUG2eUmboL5x+bOkP2MmWZjLMfFU0oo6kSOqWOLfKJk
fZztz+uLUPgvJw8PGEZ4HI1q+12Sa2OZ5qKENHDJlsEGsLsRhbbyzrU9JVs8xx+E
ZK6tpC2aU/zRAmVlA+l5caifBwGChyOWfqQX8TvbcZ4f3lWzAS5PYt3k916gfzpS
-----END RSA PRIVATE KEY-----
`

const hostPrivateKeyPassword = `sshtunnel`

func parseUserId(u string) (user, id string, err error) {
	if strings.Contains(u, "@") {
		parts := strings.Split(u, "@")
		user = parts[0]
		id = parts[1]
		if _, err = uuid.Parse(id); err == nil {
			return
		}
	}

	err = errors.Errorf("illegal format of username %s", u)
	return
}

func NewServer(sshAddr *net.TCPAddr, httpAddr *net.TCPAddr, token string) *Server {
	var server *Server

	HostPriKey, _ := ssh.ParsePrivateKeyWithPassphrase(
		[]byte(hostPrivateKey),
		[]byte(hostPrivateKeyPassword),
	)

	config := &ssh.ServerConfig{
		PasswordCallback: func(conn ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
			if token == string(password) {
				_, id, err := parseUserId(conn.User())

				if err != nil {
					return nil, err
				}

				if node := server.FindNodeById(id); node != nil {
					return nil, errors.New(fmt.Sprintf("node already exists %s", id))
				}

				return nil, nil
			}

			return nil, errors.New("invalid password")
		},
	}

	config.AddHostKey(HostPriKey)

	server = &Server{
		SshAddr:   sshAddr,
		HttpAddr:  httpAddr,
		Nodes:     list.New(),
		sshConfig: config,
	}

	return server
}

func (s *Server) Stop() {
	s.stopped = true

	if s.sshListener != nil {
		s.sshListener.Close()
	}

	s.ClearNodes()
}

func NewNode(sshConn *ssh.ServerConn) *Node {
	user, id, _ := parseUserId(sshConn.User())
	return &Node{
		Id:          id,
		Name:        user,
		RemoteIp:    sshConn.RemoteAddr().(*net.TCPAddr).IP.String(),
		ForwardList: []*Forward{},
		Heartbeat:   time.Now(),
		conn:        sshConn,
		ticker:      time.NewTicker(HeartbeatInterval),
	}
}

func (s *Server) Start() error {
	var err error
	s.sshListener, err = net.ListenTCP("tcp", s.SshAddr)
	if err != nil {
		return errors.WithStack(err)
	}

	s.httpListener, err = net.ListenTCP("tcp", s.HttpAddr)

	if err != nil {
		return errors.WithStack(err)
	}

	defer s.ClearNodes()
	defer s.sshListener.Close()
	defer s.httpListener.Close()

	go func() {
		http.Serve(s.httpListener, s.apiHandler())
	}()

	l := s.sshListener

	for {
		conn, err := l.Accept()
		if err != nil {
			if !s.stopped {
				glog.Error("accept error ", err)
				return nil
			} else {
				return err
			}
		}

		go func() {
			sshConn, channel, requests, err := ssh.NewServerConn(&Conn{
				Conn:         conn,
				ReadTimeout:  ReadTimeout,
				WriteTimeout: WriteTimeout,
			}, s.sshConfig)
			if err != nil {
				glog.Error("failed to create ssh conn ", err)
				return
			}

			glog.Infof("Connection from %s %s", sshConn.User(), sshConn.RemoteAddr())
			node := NewNode(sshConn)

			elem := s.AddNode(node)

			go s.handleRequests(requests, node)
			go s.handleChannels(channel)

			sshConn.Wait()
			glog.Infof("Disconnection from %s %s", sshConn.User(), sshConn.RemoteAddr())

			node.Clear()
			s.RemoveNode(elem)
		}()
	}

	return nil
}

func (s *Server) AddNode(n *Node) *list.Element {
	s.nodeOps.Lock()
	defer s.nodeOps.Unlock()

	elem := s.Nodes.PushBack(n)

	go func() {
		for {
			select {
			case <-n.ticker.C:
				ret, _, err := n.conn.SendRequest("keepalive", true, nil)
				if err != nil || !ret {
					n.conn.Close()
					return
				}

				glog.V(2).Infof("keep alive %s", time.Now())
				n.Heartbeat = time.Now()
			}
		}
	}()

	return elem
}

func (s *Server) RemoveNode(n *list.Element) {
	s.nodeOps.Lock()
	defer s.nodeOps.Unlock()

	s.Nodes.Remove(n)
}

func (s *Server) ListNodes() (nodes []*Node) {
	s.nodeOps.Lock()
	defer s.nodeOps.Unlock()

	for n := s.Nodes.Front(); n != nil; n = n.Next() {
		nodes = append(nodes, n.Value.(*Node))
	}

	return nodes
}

func (s *Server) FindNodeById(id string) *Node {
	s.nodeOps.Lock()
	defer s.nodeOps.Unlock()

	for n := s.Nodes.Front(); n != nil; n = n.Next() {
		if n.Value.(*Node).Id == id {
			return n.Value.(*Node)
		}
	}

	return nil
}

func (s *Server) ClearNodes() {
	s.nodeOps.Lock()
	defer s.nodeOps.Unlock()

	for e := s.Nodes.Front(); e != nil; e = e.Next() {
		e.Value.(*Node).Clear()
	}
}

func (s *Server) handleRequests(reqs <-chan *ssh.Request, node *Node) {
	for req := range reqs {
		switch req.Type {
		case ssh.NamedTcpIpForward:
			l, payload, err := ssh.HandleNamedTunnelRequest(req)
			if err != nil {
				glog.Errorf("failed to handle named tcpip forward request %s", err)
				return
			}
			s.registerAgent(l, node, payload)
		default:
			if strings.Contains(req.Type, "keepalive") {
				req.Reply(true, nil)
			} else {
				glog.Infof("request is ignored %+v", req)
				req.Reply(false, nil)
			}
		}
	}
}

func (s *Server) handleChannels(chans <-chan ssh.NewChannel) {
	for newChannel := range chans {
		newChannel.Reject(
			ssh.UnknownChannelType,
			fmt.Sprintf("unknown channel type: %s", newChannel.ChannelType()),
		)
		continue
	}
}

func (s *Server) registerAgent(listener *net.TCPListener, node *Node, msg ssh.NamedTunnelForwardMsg) {
	node.AddForwarding(msg, listener)
	go ssh.ForwardTraffic(listener, node.conn)
}
