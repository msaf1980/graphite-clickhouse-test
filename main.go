package main

import (
	"flag"
	"os"
	"path/filepath"
	"time"

	"log"
	"strings"

	"go.uber.org/zap"
)

type StringSlice []string

func (u *StringSlice) Set(value string) error {
	*u = append(*u, value)
	return nil
}

func (u *StringSlice) String() string {
	return "[ " + strings.Join(*u, ", ") + " ]"
}

var (
	logger       *zap.Logger
	now          int64
	curDir       string
	breakOnError bool
	exit         bool
	storeDir     string
)

func main() {
	var err error

	now = time.Now().Unix()

	curDir, err = filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}

	logger, err = zap.NewProduction()
	if err != nil {
		log.Fatal(err)
	}

	var tests StringSlice
	flag.Var(&tests, "config", "test dir (with test.yml and other)")
	flag.BoolVar(&breakOnError, "break", false, "Break test on error (without stop daemons for debug)")
	flag.BoolVar(&exit, "exit", false, "Exit after test (without stop clickhouse and no remove config files for debug)")
	flag.StringVar(&storeDir, "store", "/tmp/test-graphite", "Dir for store file")
	flag.Parse()

	if len(tests) == 0 {
		logger.Fatal("tests config not set")
	} else if len(tests) > 1 && exit {
		logger.Fatal("in exit mode can run only 1 test")
	}

	var testsConfig []*TestConfig
	for _, test := range tests {
		cfg, err := loadTest(test)
		if err == nil {
			testsConfig = append(testsConfig, cfg)
		} else {
			logger.Fatal("FAILED", zap.String("test", test), zap.Error(err))
		}
	}

	if err = os.Mkdir(storeDir, 0755); err != nil {
		log.Fatal(err)
	}

	var testsFailed []string
	for _, test := range testsConfig {
		logger.Info("Starting test", zap.String("config", test.Dir))
		if !runTest(test) {
			testsFailed = append(testsFailed, test.Dir)
		}
	}

	if len(testsFailed) > 0 {
		logger.Fatal("FAILED", zap.Strings("tests", testsFailed))
	} else {
		logger.Info("SUCCESS")
	}
}
