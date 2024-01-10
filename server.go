package prscd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/pilarjs/prscd/util"
	"github.com/pilarjs/prscd/websocket"
	"github.com/pilarjs/prscd/webtransport"
	"github.com/yomorun/yomo"
	"github.com/yomorun/yomo/core/router"
	"github.com/yomorun/yomo/pkg/config"
)

var log = util.Log

func StartServer() {
	// check MESH_ID env
	if os.Getenv("MESH_ID") == "" {
		log.Fatal(errors.New("env check failed"))
	}

	// DEBUG env indicates development mode, verbose log
	if os.Getenv("DEBUG") == "true" {
		log.SetLogLevel(util.DEBUG)
		log.Debug("IN DEVELOPMENT ENV")
	}

	// WITH_YOMO_ZIPPER env indicates start YOMO Zipper in this process
	if os.Getenv("WITH_YOMO_ZIPPER") == "true" {
		go startYomoZipper()
		// sleep 2 seconds to wait for YoMo Zipper ready
		time.Sleep(2 * time.Second)
	} else {
		log.Debug("Skip integrated YOMO Zipper")
	}

	// default addr and port listening
	addr := "0.0.0.0:443"
	if os.Getenv("PORT") != "" {
		addr = fmt.Sprintf("0.0.0.0:%s", os.Getenv("PORT"))
	}

	// load TLS cert and key, halt if error occurs,
	// this helped developers to find out TLS related issues asap.
	config, err := loadTLS(os.Getenv("CERT_FILE"), os.Getenv("KEY_FILE"))
	if err != nil {
		log.Fatal(err)
	}

	// start WebSocket listener
	go websocket.ListenAndServe(addr, config)

	// start WebTransport listener
	go webtransport.ListenAndServe(addr, config)

	// Ctrl-C or kill <pid> graceful shutdown
	// - `kill -SIGUSR1 <pid>` customize
	// - `kill -SIGTERM <pid>` graceful shutdown
	// - `kill -SIGUSR2 <pid>` inspect golang GC
	log.Info("creating pid file", "pid", os.Getpid())
	// write pid to ./prscd.pid, overwrite if exists
	pidFile := "./prscd.pid"
	f, err := os.OpenFile(pidFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	_, err = f.WriteString(fmt.Sprintf("%d", os.Getpid()))
	if err != nil {
		log.Fatal(err)
	}

	log.Debug(fmt.Sprintf("Prscd Dev Server is running on https://%s:%s/v1", os.Getenv("DOMAIN"), os.Getenv("PORT")))

	c := make(chan os.Signal, 1)
	registerSignal(c)
}

func startYomoZipper() {
	conf, err := config.ParseConfigFile("./yomo.yaml")
	if err != nil {
		log.Fatal(err)
	}
	log.Debug("integrated YoMo config:", "config file", conf)
	log.Debug("integrated YoMo zipper:", "zipper endpoint", fmt.Sprintf("%s:%d", conf.Host, conf.Port))

	zipper, err := yomo.NewZipper(conf.Name, router.Default(), nil)
	if err != nil {
		log.Fatal(err)
	}

	err = zipper.ListenAndServe(context.Background(), fmt.Sprintf("%s:%d", conf.Host, conf.Port))
	if err != nil {
		log.Fatal(err)
	}
}
