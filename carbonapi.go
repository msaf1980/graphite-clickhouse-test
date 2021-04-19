package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
	"text/template"

	"github.com/phayes/freeport"
)

type Carbonapi struct {
	bin        string
	configFile string
	configTpl  string
	address    string
	cmd        *exec.Cmd
}

func CarbonapiStart(bin, configTpl, testDir, chAddr, config string) (*Carbonapi, error) {
	var err error

	if len(bin) == 0 {
		return nil, fmt.Errorf("bin not set")
	}

	c := &Carbonapi{bin: bin, configTpl: configTpl}
	port, err := freeport.GetFreePort()
	if err != nil {
		return nil, err
	}
	c.address = "127.0.0.1:" + strconv.Itoa(port)

	tmpl, err := template.New(configTpl).ParseFiles(filepath.Join(testDir, configTpl))
	if err != nil {
		return nil, err
	}
	param := struct {
		ADDR     string
		GCH_ADDR string
	}{
		ADDR:     chAddr,
		GCH_ADDR: c.address,
	}

	c.configFile = config
	f, err := os.OpenFile(c.configFile, os.O_WRONLY|os.O_CREATE, 0644)
	f.Truncate(0)
	if err != nil {
		return nil, err
	}
	err = tmpl.Execute(f, param)
	if err != nil {
		return nil, err
	}

	c.cmd = exec.Command(bin, "-config", c.configFile)
	c.cmd.Stdout = os.Stdout
	c.cmd.Stderr = os.Stderr
	//c.cmd.Env = append(c.cmd.Env, "TZ=UTC")
	err = c.cmd.Start()
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (c *Carbonapi) Stop() error {
	if c.cmd == nil {
		return nil
	}
	var err error
	if err = c.cmd.Process.Kill(); err == nil {
		if err = c.cmd.Wait(); err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
					ec := status.ExitStatus()
					if ec == 0 || ec == -1 {
						return nil
					}
				}
			}
		}
	}
	return err
}

func (c *Carbonapi) Address() string {
	return c.address
}
