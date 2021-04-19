package main

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/phayes/freeport"
	"go.uber.org/zap"
)

type Clickhouse struct {
	Version string `yaml:"version"`
	Dir     string `yaml:"dir"`

	address   string `yaml:"-"`
	container string `yaml:"-"`

	docker string `yaml:"-"`
}

func (c *Clickhouse) Start(docker, dockerImage string) (error, string) {
	c.docker = docker

	if len(c.Version) == 0 {
		logger.Error("Starting clickhouse", zap.String("dir", c.Dir), zap.String("version", c.Version))
		return fmt.Errorf("version not set"), ""
	}
	port, err := freeport.GetFreePort()
	if err != nil {
		logger.Error("Starting clickhouse", zap.String("dir", c.Dir), zap.String("version", c.Version))
		return err, ""
	}

	c.address = "127.0.0.1:" + strconv.Itoa(port)
	c.container = "graphite-test-clickhouse"

	chStart := []string{"run", "-d",
		"--name", c.container,
		"--ulimit", "nofile=262144:262144",
		"-p", c.address + ":8123",
		//"-e", "TZ=UTC",
		"-v", curDir + "/tests/" + c.Dir + "/config.xml:/etc/clickhouse-server/config.xml",
		"-v", curDir + "/tests/" + c.Dir + "/users.xml:/etc/clickhouse-server/users.xml",
		"-v", curDir + "/tests/" + c.Dir + "/rollup.xml:/etc/clickhouse-server/config.d/rollup.xml",
		"-v", curDir + "/tests/" + c.Dir + "/init.sql:/docker-entrypoint-initdb.d/init.sql",
		dockerImage + ":" + c.Version,
	}

	logger.Info("Starting clickhouse",
		zap.String("dir", c.Dir),
		zap.String("version", c.Version),
		zap.String("command", docker+" "+strings.Join(chStart, " ")),
	)

	cmd := exec.Command(docker, chStart...)
	stdoutStderr, err := cmd.CombinedOutput()

	return err, string(stdoutStderr)
}

func (c *Clickhouse) Stop(delete bool) (error, string) {
	if len(c.container) == 0 {
		return nil, ""
	}

	chStop := []string{"stop", c.container}

	cmd := exec.Command(c.docker, chStop...)
	stdoutStderr, err := cmd.CombinedOutput()

	if err == nil && delete {
		return c.Delete()
	}
	return err, string(stdoutStderr)
}

func (c *Clickhouse) Delete() (error, string) {
	if len(c.container) == 0 {
		return nil, ""
	}

	chDel := []string{"rm", c.container}

	cmd := exec.Command(c.docker, chDel...)
	stdoutStderr, err := cmd.CombinedOutput()

	return err, string(stdoutStderr)
}

func (c *Clickhouse) Address() string {
	return c.address
}

func (c *Clickhouse) Container() string {
	return c.container
}
