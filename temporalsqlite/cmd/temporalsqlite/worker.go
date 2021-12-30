package main

import (
	"fmt"

	"github.com/cretz/temporal-sdk-go-advanced/temporalsqlite"
	"github.com/urfave/cli/v2"
	"go.temporal.io/sdk/worker"
)

type workerConfig struct {
	commonConfig
	options temporalsqlite.SqliteWorkerOptions
}

func (w *workerConfig) flags() []cli.Flag {
	return append(w.commonConfig.flags(),
		&cli.IntFlag{
			Name:        "default-requests-until-continue-as-new",
			Usage:       "Default number of requests until continue-as-new",
			Value:       temporalsqlite.DefaultRequestsUntilContinueAsNew,
			Destination: &w.options.DefaultRequestsUntilContinueAsNew,
		},
		&cli.BoolFlag{
			Name:        "log-queries",
			Usage:       "Whether to log queries",
			Destination: &w.options.LogQueries,
		},
		&cli.BoolFlag{
			Name:        "log-actions",
			Usage:       "Whether to log SQLite actions",
			Destination: &w.options.LogActions,
		},
		&cli.BoolFlag{
			Name:        "ignore-update-errors",
			Usage:       "Whether to ignore update errors instead of failing workflow",
			Destination: &w.options.IgnoreUpdateErrors,
		},
	)
}

func workerCmd() *cli.Command {
	var config workerConfig
	return &cli.Command{
		Name:  "worker",
		Usage: "run a SQLite worker",
		Flags: config.flags(),
		Action: func(ctx *cli.Context) error {
			if ctx.Args().Len() > 0 {
				return fmt.Errorf("no arguments allowed")
			}
			c, err := config.dialClient()
			if err != nil {
				return err
			}
			defer c.Close()
			w := worker.New(c, config.taskQueue, worker.Options{})
			temporalsqlite.RegisterSqliteWorker(w, config.options)
			return w.Run(worker.InterruptCh())
		},
	}
}
