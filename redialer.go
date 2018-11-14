package adslproxy

import (
	"context"
	"github.com/golang/glog"
	"github.com/pkg/errors"
	"os/exec"
	"regexp"
	"runtime"
	"time"
)

func runWinCommand(to time.Duration, name string, arg ...string) ([]byte, error) {
	ctx := context.Background()
	ctx, _ = context.WithTimeout(ctx, to)
	cmd := exec.CommandContext(ctx, name, arg...)
	output, err := cmd.CombinedOutput()

	return output, err
}

const RASDIAL = "rasdial"

func winConnect(to time.Duration, name, username, password string) error {
	output, err := runWinCommand(to, RASDIAL, name, username, password)
	if err != nil {
		return err
	}

	glog.Infof("adsl connect result :%s", string(output))
	return nil
}

func winDisconnect(to time.Duration, name string) error {
	output, err := runWinCommand(to, RASDIAL, name, "/DISCONNECT")
	if err != nil {
		return err
	}

	glog.Infof("adsl disconnect result :%s", string(output))
	return nil
}

type Redialer interface {
	Redial(config *AdslConfig) error
}

type WindowsRedialer struct {
}

func (wr *WindowsRedialer) Redial(c *AdslConfig) error {
	err := winDisconnect(10*time.Second, c.InterfaceName)
	if err != nil {
		return err
	}

	err = winConnect(10*time.Second, c.InterfaceName, c.Username, c.Password)
	return err
}

const DefaultPppoeStartScript = "/usr/sbin/pppoe-start"
const DefaultPppoeStopScript = "/usr/sbin/pppoe-stop"

func runBashScript(script, desc string) error {
	cmd := exec.Command("/bin/bash", script)
	if _, err := cmd.Output(); err != nil {
		return err
	} else {
		return nil
	}
}

func checkRoute() error {
	cmd := exec.Command("/usr/sbin/ip", "route")
	var output []byte
	var err error
	if output, err = cmd.Output(); err != nil {
		return err
	}

	if regexp.MustCompile("ppp\\d+").Match(output) {
		return nil
	}

	cmd = exec.Command("/usr/sbin/ip", "link")
	if output, err = cmd.Output(); err != nil {
		return err
	}

	ret := regexp.MustCompile("ppp\\d+").Find(output)
	if len(ret) == 0 {
		return errors.New("no ppp dev found")
	}

	cmd = exec.Command("/usr/sbin/route", "add", "default", "dev", string(ret))
	if output, err = cmd.Output(); err != nil {
		return err
	}

	return nil
}

type CentosRedialer struct {
}

func (wr *CentosRedialer) Redial(_ *AdslConfig) error {
	runBashScript(DefaultPppoeStopScript, "Stop pppoe")

	if err := runBashScript(DefaultPppoeStartScript, "Start pppoe"); err != nil {
		return err
	}

	return checkRoute()
}

type AdslConfig struct {
	// used in windows
	Username      string
	Password      string
	InterfaceName string

	RedialInterval time.Duration
}

func (ac *AdslConfig) Redial() error {
	switch runtime.GOOS {
	case "windows":
		return (&WindowsRedialer{}).Redial(ac)
	case "linux":
		return (&CentosRedialer{}).Redial(ac)
	default:
		return errors.Errorf("Not supported os %s", runtime.GOOS)
	}
}
