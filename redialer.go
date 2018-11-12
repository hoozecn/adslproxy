package adslproxy

import (
	"github.com/pkg/errors"
	"os/exec"
	"regexp"
	"runtime"
	"time"
)

type Redialer interface {
	Redial() error
}

type WindowsRedialer struct {
	username      string
	password      string
	interfaceName string
}

func (wr *WindowsRedialer) Redial() error {
	return nil
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

func (wr *CentosRedialer) Redial() error {
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
		dialer := &WindowsRedialer{
			ac.Username,
			ac.Password,
			ac.InterfaceName,
		}

		return dialer.Redial()
	case "linux":
		dialer := &CentosRedialer{}
		return dialer.Redial()
	default:
		return errors.Errorf("Not supported os %s", runtime.GOOS)
	}
}
