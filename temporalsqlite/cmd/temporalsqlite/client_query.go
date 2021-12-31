package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/cretz/temporal-sdk-go-advanced/temporalsqlite"
	"github.com/olekukonko/tablewriter"
	"github.com/urfave/cli/v2"
	"go.temporal.io/sdk/client"
)

type queryConfig struct {
	clientConfig
	params cli.StringSlice
	multi  bool
}

func (q *queryConfig) flags() []cli.Flag {
	return append(q.clientConfig.flags(),
		&cli.StringSliceFlag{
			Name:        "param",
			Usage:       "Parameters in the form of <index|name>=<type>(<value>) (or just =null, indices start at 1)",
			Destination: &q.params,
		},
		&cli.BoolFlag{
			Name:        "multi",
			Usage:       "Treat the query as a script with multiple statements (cannot have params)",
			Destination: &q.multi,
		},
	)
}

var errBadParam = errors.New("all parameters must be in the form of <index|name>=<type>(<value>) (or just =null)")

func (q *queryConfig) toStmt(args cli.Args) (*temporalsqlite.Stmt, error) {
	if args.Len() == 0 || args.Len() > 1 {
		return nil, fmt.Errorf("must have a single query argument")
	} else if q.multi && len(q.params.Value()) > 0 {
		return nil, fmt.Errorf("cannot have parameters on multi statement")
	}
	stmt := &temporalsqlite.Stmt{Query: args.First(), Multi: q.multi}

	for _, param := range q.params.Value() {
		pieces := strings.SplitN(param, "=", 2)
		if len(pieces) != 2 {
			return nil, errBadParam
		}

		// Convert val type
		var val interface{}
		if pieces[1] != "null" {
			valPieces := strings.SplitN(pieces[1], "(", 2)
			if len(valPieces) != 2 || !strings.HasSuffix(valPieces[1], ")") {
				return nil, errBadParam
			}
			valPieces[1] = strings.TrimSuffix(valPieces[1], ")")
			var err error
			switch valPieces[0] {
			case "int":
				val, err = strconv.ParseInt(valPieces[1], 10, 64)
			case "float":
				val, err = strconv.ParseFloat(valPieces[1], 64)
			case "string":
				val = valPieces[1]
			case "bytes":
				val = []byte(valPieces[1])
			default:
				return nil, fmt.Errorf("parameter type was %q, expected int, float, string or bytes", valPieces[0])
			}
			if err != nil {
				return nil, fmt.Errorf("invalid parameter value %q: %v", valPieces[1], err)
			}
		}

		// Use index if num, otherwise named
		if i, err := strconv.Atoi(pieces[0]); err == nil {
			if stmt.IndexedParams == nil {
				stmt.IndexedParams = map[int]interface{}{}
			}
			stmt.IndexedParams[i] = val
		} else {
			if stmt.NamedParams == nil {
				stmt.NamedParams = map[string]interface{}{}
			}
			stmt.NamedParams[pieces[0]] = val
		}
	}
	return stmt, nil
}

func dialAndConnectWithStmt(
	ctx *cli.Context,
	config queryConfig,
) (cl client.Client, dbCl *temporalsqlite.Client, stmt *temporalsqlite.Stmt, err error) {
	cl, err = config.dialClient()
	defer func() {
		if cl != nil && err != nil {
			cl.Close()
		}
	}()
	if err == nil {
		dbCl, err = config.connect(ctx.Context, cl)
	}
	if err == nil {
		stmt, err = config.toStmt(ctx.Args())
	}
	return
}

func dumpSuccesses(successes []*temporalsqlite.StmtResultSuccess) {
	var dumpedFirst bool
	for _, success := range successes {
		if len(success.Rows) == 0 {
			continue
		} else if dumpedFirst {
			fmt.Println()
		} else {
			dumpedFirst = true
		}
		dumpSuccess(success)
	}
}

func dumpSuccess(s *temporalsqlite.StmtResultSuccess) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader(s.ColumnNames)
	for _, row := range s.Rows {
		rowStrs := make([]string, len(row))
		for i, val := range row {
			rowStrs[i] = fmt.Sprintf("%v", val)
		}
		table.Append(rowStrs)
	}
	table.Render()
}

func queryCmd() *cli.Command {
	var config queryConfig
	return &cli.Command{
		Name:  "query",
		Usage: "send a read-only query and get a response",
		Flags: config.flags(),
		Action: func(ctx *cli.Context) error {
			client, dbClient, stmt, err := dialAndConnectWithStmt(ctx, config)
			if err != nil {
				return err
			}
			defer client.Close()
			defer dbClient.Close()

			// Query and dump
			res, err := dbClient.Query(ctx.Context, stmt)
			if err != nil {
				return err
			}
			dumpSuccesses(res[0].Successes)
			if res[0].Error != nil {
				return res[0].Error
			}
			log.Printf("Queries succeeded")
			return nil
		},
	}
}

func updateCmd() *cli.Command {
	var config queryConfig
	return &cli.Command{
		Name:  "update",
		Usage: "send an update that, by default, fails the entire DB if it does not succeed",
		Flags: config.flags(),
		Action: func(ctx *cli.Context) error {
			client, dbClient, stmt, err := dialAndConnectWithStmt(ctx, config)
			if err != nil {
				return err
			}
			defer client.Close()
			defer dbClient.Close()

			// Update
			if err := dbClient.Update(ctx.Context, stmt); err != nil {
				return err
			}
			log.Printf("Update sent")
			return nil
		},
	}
}

func execCmd() *cli.Command {
	var config queryConfig
	return &cli.Command{
		Name:  "exec",
		Usage: "exec a query and get a response",
		Flags: config.flags(),
		Action: func(ctx *cli.Context) error {
			client, dbClient, stmt, err := dialAndConnectWithStmt(ctx, config)
			if err != nil {
				return err
			}
			defer client.Close()
			defer dbClient.Close()

			// Exec and dump
			res, err := dbClient.Exec(ctx.Context, stmt)
			if err != nil {
				return err
			}
			dumpSuccesses(res[0].Successes)
			if res[0].Error != nil {
				return res[0].Error
			}
			log.Printf("Executions succeeded")
			return nil
		},
	}
}
