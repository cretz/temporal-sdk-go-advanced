package main

import (
	"context"
	"fmt"
	"log"

	"crawshaw.io/sqlite"
	"github.com/cretz/temporal-sdk-go-advanced/temporalsqlite"
	"github.com/urfave/cli/v2"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
)

type clientConfig struct {
	commonConfig
	db              string
	createIfStopped bool
}

func (c *clientConfig) flags() []cli.Flag {
	return append(c.commonConfig.flags(),
		&cli.StringFlag{
			Name:        "db",
			Usage:       "The database name",
			Required:    true,
			Destination: &c.db,
		},
		&cli.BoolFlag{
			Name:        "create-if-stopped",
			Usage:       "Create the DB if it has stopped",
			Destination: &c.createIfStopped,
		},
	)
}

func (c *clientConfig) connect(ctx context.Context, cl client.Client) (*temporalsqlite.Client, error) {
	var opts temporalsqlite.ConnectDBOptions
	opts.StartWorkflow.ID = c.db
	opts.StartWorkflow.TaskQueue = c.taskQueue
	if c.createIfStopped {
		opts.StartWorkflow.WorkflowIDReusePolicy = enums.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE_FAILED_ONLY
	}
	return temporalsqlite.ConnectDB(ctx, cl, opts)
}

func (c *clientConfig) newClient(cl client.Client) (*temporalsqlite.Client, error) {
	return temporalsqlite.NewClient(cl, c.db)
}

func getOrCreateCmd() *cli.Command {
	var config clientConfig
	return &cli.Command{
		Name:  "get-or-create",
		Usage: "get or create the DB",
		Flags: config.flags(),
		Action: func(ctx *cli.Context) error {
			if ctx.Args().Len() > 0 {
				return fmt.Errorf("no arguments allowed")
			}
			// Dial
			client, err := config.dialClient()
			if err != nil {
				return err
			}
			defer client.Close()

			// Connect which will do a create
			dbClient, err := config.connect(ctx.Context, client)
			if err != nil {
				return err
			}
			defer dbClient.Close()

			// Describe
			resp, err := client.DescribeWorkflowExecution(ctx.Context, config.db, "")
			if err != nil {
				return err
			}
			log.Printf("DB %v status: %v", config.db, resp.WorkflowExecutionInfo.Status)

			// Show failure is failed
			if resp.WorkflowExecutionInfo.Status == enums.WORKFLOW_EXECUTION_STATUS_FAILED {
				run, err := dbClient.GetRun(ctx.Context)
				if err != nil {
					return err
				}
				log.Printf("Failure: %v", run.Get(ctx.Context))
			}
			return nil
		},
	}
}

func stopCmd() *cli.Command {
	var config clientConfig
	return &cli.Command{
		Name:  "stop",
		Usage: "stop the DB",
		Flags: config.flags(),
		Action: func(ctx *cli.Context) error {
			if ctx.Args().Len() > 0 {
				return fmt.Errorf("no arguments allowed")
			}
			// Dial
			client, err := config.dialClient()
			if err != nil {
				return err
			}
			defer client.Close()

			// Get client and attempt stop
			dbClient, err := config.newClient(client)
			if err != nil {
				return err
			}
			defer dbClient.Close()
			if err := dbClient.StopDB(ctx.Context); err != nil {
				return err
			}
			log.Printf("DB stopped")
			return nil
		},
	}
}

type saveConfig struct {
	clientConfig
	file string
}

func (s *saveConfig) flags() []cli.Flag {
	return append(s.clientConfig.flags(),
		&cli.StringFlag{
			Name:        "file",
			Usage:       "File to put DB at",
			Value:       "sqlite.db",
			Destination: &s.file,
		},
	)
}

func saveCmd() *cli.Command {
	var config saveConfig
	return &cli.Command{
		Name:  "save",
		Usage: "download the DB into a local SQLite file",
		Flags: config.flags(),
		Action: func(ctx *cli.Context) error {
			if ctx.Args().Len() > 0 {
				return fmt.Errorf("no arguments allowed")
			}
			// Dial
			client, err := config.dialClient()
			if err != nil {
				return err
			}
			defer client.Close()

			// Get client and serialize
			dbClient, err := config.newClient(client)
			if err != nil {
				return err
			}
			defer dbClient.Close()
			b, err := dbClient.Serialize(ctx.Context)
			if err != nil {
				return err
			}

			// Deserialize into new DB and backup to disk
			conn, err := sqlite.OpenConn(":memory:")
			if err != nil {
				return err
			}
			defer conn.Close()
			if err := conn.Deserialize(sqlite.NewSerialized("", b, false)); err != nil {
				return err
			}
			dstConn, err := conn.BackupToDB("", config.file)
			if err != nil {
				return err
			} else if err = dstConn.Close(); err != nil {
				return err
			}
			log.Printf("DB saved at %v", config.file)
			return nil
		},
	}
}
