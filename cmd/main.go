package main

import (
	"log"
	"os"
	"time"

	"github.com/urfave/cli"
	"github.com/zionnode/xscar"
)

var SYNC_TIME int

func main() {
	app := cli.NewApp()
	app.Name = "xscar"
	app.Usage = "sidecar for Xray"
	app.Version = "0.0.11"
	app.Author = "Ehco1996"

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "grpc-endpoint, gp",
			Value:       "127.0.0.1:8080",
			Usage:       "V2ray开放的GRPC地址",
			EnvVar:      "XSCAR_GRPC_ENDPOINT",
			Destination: &xscar.GRPC_ENDPOINT,
		},
		cli.StringFlag{
			Name:        "api-endpoint, ap",
			Value:       "http://fun.com/api",
			Usage:       "django-sspanel开放的Vemss Node Api地址",
			EnvVar:      "XSCAR_API_ENDPOINT",
			Destination: &xscar.API_ENDPOINT,
		},
		cli.IntFlag{
			Name:        "sync-time, st",
			Value:       60,
			Usage:       "与django-sspanel同步的时间间隔",
			EnvVar:      "XSCAR_SYNC_TIME",
			Destination: &SYNC_TIME,
		},
	}

	app.Action = func(c *cli.Context) error {
		up := xscar.NewUserPool()

		// 立即执行第一次同步
		go xscar.SyncTask(up)

		tick := time.Tick(time.Duration(SYNC_TIME) * time.Second)
		for {
			<-tick
			go xscar.SyncTask(up)
		}
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
