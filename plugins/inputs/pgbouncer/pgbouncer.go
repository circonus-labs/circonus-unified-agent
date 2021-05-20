package pgbouncer

import (
	"bytes"
	"context"
	"fmt"
	"strconv"

	"github.com/circonus-labs/circonus-unified-agent/cua"
	"github.com/circonus-labs/circonus-unified-agent/internal"
	"github.com/circonus-labs/circonus-unified-agent/plugins/inputs"
	"github.com/circonus-labs/circonus-unified-agent/plugins/inputs/postgresql"
	_ "github.com/jackc/pgx/stdlib" // register driver
)

type PgBouncer struct {
	postgresql.Service
}

var ignoredColumns = map[string]bool{"user": true, "database": true, "pool_mode": true,
	"avg_req": true, "avg_recv": true, "avg_sent": true, "avg_query": true,
}

var sampleConfig = `
  ## specify address via a url matching:
  ##   postgres://[pqgotest[:password]]@localhost[/dbname]\
  ##       ?sslmode=[disable|verify-ca|verify-full]
  ## or a simple string:
  ##   host=localhost user=pqgotest password=... sslmode=... dbname=app_production
  ##
  ## All connection parameters are optional.
  ##
  address = "host=localhost user=pgbouncer sslmode=disable"
`

func (p *PgBouncer) SampleConfig() string {
	return sampleConfig
}

func (p *PgBouncer) Description() string {
	return "Read metrics from one or many pgbouncer servers"
}

func (p *PgBouncer) Gather(ctx context.Context, acc cua.Accumulator) error {
	var (
		err     error
		query   string
		columns []string
	)

	query = `SHOW STATS`

	rows, err := p.DB.Query(query)
	if err != nil {
		return fmt.Errorf("db query (%s): %w", query, err)
	}

	defer rows.Close()

	// grab the column information from the result
	if columns, err = rows.Columns(); err != nil {
		return fmt.Errorf("row columns: %w", err)
	}

	for rows.Next() {
		tags, columnMap, err := p.parseRow(rows, columns)

		if err != nil {
			return err
		}

		fields := make(map[string]interface{})
		for col, val := range columnMap {
			_, ignore := ignoredColumns[col]
			if ignore {
				continue
			}

			switch v := (*val).(type) {
			case int64:
				// Integer fields are returned in pgbouncer 1.5 through 1.9
				fields[col] = v
			case string:
				// Integer fields are returned in pgbouncer 1.12
				integer, err := strconv.ParseInt(v, 10, 64)
				if err != nil {
					return fmt.Errorf("parseint (%s): %w", v, err)
				}

				fields[col] = integer
			}
		}
		acc.AddFields("pgbouncer", fields, tags)
	}

	err = rows.Err()
	if err != nil {
		return fmt.Errorf("rows err: %w", err)
	}

	query = `SHOW POOLS`

	poolRows, err := p.DB.Query(query)
	if err != nil {
		return fmt.Errorf("db query (%s): %w", query, err)
	}

	defer poolRows.Close()

	// grab the column information from the result
	if columns, err = poolRows.Columns(); err != nil {
		return fmt.Errorf("row columns: %w", err)
	}

	for poolRows.Next() {
		tags, columnMap, err := p.parseRow(poolRows, columns)
		if err != nil {
			return err
		}

		if user, ok := columnMap["user"]; ok {
			if s, ok := (*user).(string); ok && s != "" {
				tags["user"] = s
			}
		}

		if poolMode, ok := columnMap["pool_mode"]; ok {
			if s, ok := (*poolMode).(string); ok && s != "" {
				tags["pool_mode"] = s
			}
		}

		fields := make(map[string]interface{})
		for col, val := range columnMap {
			_, ignore := ignoredColumns[col]
			if !ignore {
				fields[col] = *val
			}
		}
		acc.AddFields("pgbouncer_pools", fields, tags)
	}

	return poolRows.Err()
}

type scanner interface {
	Scan(dest ...interface{}) error
}

func (p *PgBouncer) parseRow(row scanner, columns []string) (map[string]string, map[string]*interface{}, error) {
	var columnVars []interface{}
	var dbname bytes.Buffer

	// this is where we'll store the column name with its *interface{}
	columnMap := make(map[string]*interface{})

	for _, column := range columns {
		columnMap[column] = new(interface{})
	}

	// populate the array of interface{} with the pointers in the right order
	for i := 0; i < len(columnMap); i++ {
		columnVars = append(columnVars, columnMap[columns[i]])
	}

	// deconstruct array of variables and send to Scan
	err := row.Scan(columnVars...)
	if err != nil {
		return nil, nil, fmt.Errorf("row scan: %w", err)
	}
	if columnMap["database"] != nil {
		// extract the database name from the column map
		dbname.WriteString((*columnMap["database"]).(string))
	} else {
		dbname.WriteString("postgres")
	}

	var tagAddress string
	tagAddress, err = p.SanitizedAddress()
	if err != nil {
		return nil, nil, fmt.Errorf("sanitize addr: %w", err)
	}

	// Return basic tags and the mapped columns
	return map[string]string{"server": tagAddress, "db": dbname.String()}, columnMap, nil
}

func init() {
	inputs.Add("pgbouncer", func() cua.Input {
		return &PgBouncer{
			Service: postgresql.Service{
				MaxIdle: 1,
				MaxOpen: 1,
				MaxLifetime: internal.Duration{
					Duration: 0,
				},
				IsPgBouncer: true,
			},
		}
	})
}
