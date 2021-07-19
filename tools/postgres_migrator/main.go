package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"syscall"

	"cloud.google.com/go/bigquery"
	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/pkg/errors"
	"golang.org/x/term"
	"google.golang.org/api/iterator"
)

type sqlCommand string

func (s sqlCommand) Format(v ...interface{}) string {
	return fmt.Sprintf(string(s), v...)
}

var ping sqlCommand = "SELECT 1;"

var createTable sqlCommand = `
CREATE TABLE IF NOT EXISTS %s (
    metadata JSON,
    scenario JSON,
    latencies JSON,
    clientStats JSON,
    serverStats JSON,
    serverCores JSON,
    summary JSON,
    clientSuccess JSON,
    serverSuccess JSON,
    requestResults JSON,
    serverCpuStats JSON,
    serverCpuUsage FLOAT
);`

var selectScenarios sqlCommand = "SELECT * FROM `%s`.%s.%s LIMIT 1;"

var gcpAuthError = `Please authenticate your workstation and try again:

To use a service account, set the $GOOGLE_APPLICATION_CREDENTIALS
environment variable to the path for its JSON key.

To use your own user credentials, please perform the following steps:

  1. Run: gcloud auth application-default login

  2. Follow its instructions using your web browser

  3. Copy the path where the credentials are saved

  4. Export the $GOOGLE_APPLICATIONS_CREDENTIALS environment variable
     set to this path.`

var printf = fmt.Printf

var println = fmt.Println

func printAndAbortf(err error, messageFmt string, v ...interface{}) {
	fmt.Printf(messageFmt+": %v\n", append(v, err))
	os.Exit(1)
}

type ResultRow struct {
	metadata       bigquery.Value
	scenario       bigquery.Value
	latencies      bigquery.Value
	clientStats    bigquery.Value
	serverStats    bigquery.Value
	summary        bigquery.Value
	serverCPUStats bigquery.Value
	serverCores    bigquery.Value
	clientSuccess  bigquery.Value
	serverSuccess  bigquery.Value
	requestResults bigquery.Value
	serverCPUUsage bigquery.Value
}

func newResultRow(row map[string]bigquery.Value) (*ResultRow, error) {
	resultRow := &ResultRow{}

	metadata, err := json.Marshal(row["metadata"])
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse and marshal JSON for metadata record in BigQuery row")
	}
	resultRow.metadata = metadata

	scenario, err := json.Marshal(row["scenario"])
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse and marshal JSON for scenario record in BigQuery row")
	}
	resultRow.scenario = scenario

	latencies, err := json.Marshal(row["latencies"])
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse and marshal JSON for latencies record in BigQuery row")
	}
	resultRow.latencies = latencies

	clientStats, err := json.Marshal(row["clientStats"])
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse and marshal JSON for clientStats record in BigQuery row")
	}
	resultRow.clientStats = clientStats

	serverStats, err := json.Marshal(row["serverStats"])
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse and marshal JSON for serverStats record in BigQuery row")
	}
	resultRow.serverStats = serverStats

	summary, err := json.Marshal(row["summary"])
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse and marshal JSON for summary record in BigQuery row")
	}
	resultRow.summary = summary

	serverCPUStats, err := json.Marshal(row["serverCPUStats"])
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse and marshal JSON for summary record in BigQuery row")
	}
	resultRow.serverCPUStats = serverCPUStats

	resultRow.serverCores = fmt.Sprintf("%v", row["serverCores"])
	resultRow.clientSuccess = fmt.Sprintf("%v", row["clientSuccess"])
	resultRow.serverSuccess = fmt.Sprintf("%v", row["serverSuccess"])
	resultRow.requestResults = fmt.Sprintf("%v", row["requestResults"])
	resultRow.serverCPUUsage = row["serverCpuUsage"]

	return resultRow, nil
}

func addRowToPostgres(ctx context.Context, psql *sql.DB, table string, result *ResultRow) error {
	res, err := psql.ExecContext(ctx, fmt.Sprintf(
		`INSERT INTO %s
(metadata, scenario, latencies, clientStats, serverStats, serverCores, summary, clientSuccess, serverSuccess, requestResults, serverCpuStats, serverCpuUsage)
VALUES ('%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', %v);`,
		table,
		result.metadata,
		result.scenario,
		result.latencies,
		result.clientStats,
		result.serverStats,
		result.serverCores,
		result.summary,
		result.clientSuccess,
		result.serverSuccess,
		result.requestResults,
		result.serverCPUStats,
		result.serverCPUUsage,
	))
	if err != nil {
		return errors.Wrapf(err, "insert into PostgreSQL table failed")
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return errors.Wrapf(err, "could not access rows affected")
	}
	if rows != 1 {
		return errors.Wrapf(err, "expected 1 row to be affected, but %d rows were changed", rows)
	}

	return nil
}

func main() {
	var bqProject string
	var bqLocation string
	var bqDataset string
	var bqTable string
	var psqlHost string
	var psqlPort string
	var psqlUser string
	var psqlDatabase string
	var psqlTable string

	flag.StringVar(&bqProject, "bq-project", "", "name of the GCP project")
	flag.StringVar(&bqLocation, "bq-location", "US", "country code of the BigQuery dataset")
	flag.StringVar(&bqDataset, "bq-dataset", "", "name of the BigQuery dataset")
	flag.StringVar(&bqTable, "bq-table", "", "name of the BigQuery table")
	flag.StringVar(&psqlHost, "psql-host", "127.0.0.1", "hostname for PostgreSQL database")
	flag.StringVar(&psqlPort, "psql-port", "5432", "port for PostgreSQL database")
	flag.StringVar(&psqlUser, "psql-user", "", "username for PostgreSQL database")
	flag.StringVar(&psqlDatabase, "psql-database", "", "name of PostgreSQL database")
	flag.StringVar(&psqlTable, "psql-table", "", "name of the PostgreSQL table")
	flag.Parse()

	// search for gcloud credentials
	if _, ok := os.LookupEnv("GOOGLE_APPLICATION_CREDENTIALS"); !ok {
		// give users instructions and abort
		printf(gcpAuthError)
		os.Exit(1)
	}

	password, ok := os.LookupEnv("PSQL_PASSWORD")
	if !ok {
		// ask for the user password separately, so it is not saved in command history
		printf("PostgreSQL Password: ")
		passwordBytes, err := term.ReadPassword(syscall.Stdin)
		if err != nil {
			printAndAbortf(err, "failed to read password")
		}
		println()
		password = string(passwordBytes)
	}

	printf("Connecting to PostgreSQL... ")

	psqlURI := fmt.Sprintf("host=%s user=%s password=%s port=%s database=%s", psqlHost, psqlUser, password, psqlPort, psqlDatabase)
	psql, err := sql.Open("pgx", psqlURI)
	if err != nil {
		printAndAbortf(err, "failed to create PostgreSQL database connection")
	}
	defer psql.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err = psql.PingContext(ctx); err != nil {
		printAndAbortf(err, "unable to ping the PostgreSQL database")
	}

	println("OK.")

	printf("Connecting to BigQuery... ")

	bq, err := bigquery.NewClient(ctx, bqProject)
	if err != nil {
		printAndAbortf(err, "failed to connect to BigQuery database")
	}
	defer bq.Close()

	bqQuery := bq.Query(ping.Format())
	bqQuery.Location = bqLocation
	bqJob, err := bqQuery.Run(ctx)
	if err != nil {
		printAndAbortf(err, "failed to start BigQuery ping")
	}
	bqJobStatus, err := bqJob.Wait(ctx)
	if err != nil {
		printAndAbortf(err, "failed to wait for BigQuery ping")
	}
	if err = bqJobStatus.Err(); err != nil {
		printAndAbortf(err, "failed to return BigQuery ping result")
	}

	println("OK.")

	printf("Creating table %q if it does not exist... ", psqlTable)
	_, err = psql.ExecContext(ctx, createTable.Format(psqlTable))
	if err != nil {
		printAndAbortf(err, "failed to create PostgreSQL table")
	}
	println("OK.")

	printf("Transferring data from BigQuery to PostgreSQL... ")
	bqQuery = bq.Query(selectScenarios.Format(bqProject, bqDataset, bqTable))
	bqQuery.Location = bqLocation
	bqJob, err = bqQuery.Run(ctx)
	if err != nil {
		printAndAbortf(err, "failed to load scenarios from BigQuery (project %q, dataset %q, table %q)", bqProject, bqDataset, bqTable)
	}
	bqJobStatus, err = bqJob.Wait(ctx)
	if err != nil {
		printAndAbortf(err, "failed to wait for BigQuery scenarios")
	}
	if err = bqJobStatus.Err(); err != nil {
		printAndAbortf(err, "failed to return BigQuery scenarios result")
	}
	it, err := bqJob.Read(ctx)
	successes, failures := 0, 0
	for {
		row := make(map[string]bigquery.Value)
		err := it.Next(&row)
		if err == iterator.Done {
			break
		}
		if err != nil {
			printAndAbortf(err, "failed to read BigQuery scenario")
		}
		resultRow, err := newResultRow(row)
		if err != nil {
			printAndAbortf(err, "failed to parse BigQuery row")
		}
		if err = addRowToPostgres(ctx, psql, psqlTable, resultRow); err != nil {
			printf("BigQuery row could not be parsed and inserted: %v, problematic row: %v", err, row)
			failures++
		} else {
			successes++
		}
	}
	println("OK.")
	printf("Transferred %d/%d records.", successes, successes+failures)
}
