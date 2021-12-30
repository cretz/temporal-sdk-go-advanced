package main

import (
	"log"
	"os"

	"github.com/urfave/cli/v2"
	"go.temporal.io/sdk/client"
)

func main() {
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

var app = &cli.App{
	Commands: []*cli.Command{
		workerCmd(),
		getOrCreateCmd(),
		stopCmd(),
		saveCmd(),
		queryCmd(),
		updateCmd(),
		execCmd(),
	},
}

type commonConfig struct {
	server    string
	namespace string
	taskQueue string
}

func (c *commonConfig) flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "server",
			Usage:       "The host:port of the server",
			Value:       client.DefaultHostPort,
			Destination: &c.server,
		},
		&cli.StringFlag{
			Name:        "namespace",
			Usage:       "The namespace to use",
			Value:       client.DefaultNamespace,
			Destination: &c.namespace,
		},
		&cli.StringFlag{
			Name:        "task-queue",
			Usage:       "Task queue to use",
			Value:       "temporalsqlite-task-queue",
			Destination: &c.taskQueue,
		},
	}
}

func (c *commonConfig) dialClient() (client.Client, error) {
	return client.NewClient(client.Options{
		HostPort:  c.server,
		Namespace: c.namespace,
		Logger:    noopLog{},
	})
}

type noopLog struct{}

func (noopLog) Debug(string, ...interface{}) {}
func (noopLog) Info(string, ...interface{})  {}
func (noopLog) Warn(string, ...interface{})  {}
func (noopLog) Error(string, ...interface{}) {}
