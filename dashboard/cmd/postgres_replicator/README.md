# Postgres replicator

This tool replicates streaming data from BigQuery into PostgreSQL.

## Configuration

It is configured with a YAML file. Here is an example configuration file:

```yaml
# Source database settings
bigQuery:
  projectID: ${BQ_PROJECT_ID}

# Destination database settings
postgres:
  dbHost: 172.17.0.1
  dbPort: 5432
  dbUser: ${PG_USER}
  dbPass: ${PG_PASS}
  dbName: ${PG_DATABASE}

# Tables to tranfer from BigQuery to PostgreSQL
transfer:
  datasets:
  - name: datasetExampleName1
    tables:
      - name: tableExample1
        dateField: timeCreated
      - name: tableExample2
        dateField: timeCreated
  - name: datasetExampleName2
    tables:
      - name: tableExample3
        dateField: timeCreated
```

`BQ_PROJECT_ID`: The GCP project ID where the BigQuery instance resides. This is
available on the homepage of every GCP project, in the "Project info" card.
`PG_USER`: A user of the PostgreSQL database.
`PG_PASS`: The password associated with the above user. This can be specified as
an environment variable and will override the configuration file if set.
`PG_DATABASE`: The replication destination database.

The transfer section of the configuration file details which tables to replicate
from BigQuery. To replicate streaming data efficiently, this tool requires that
a column in the BigQuery source table store the approximate time the data was
created or added to the database. This column must be of the BigQuery
`TIMESTAMP` datatype and should only increase in value for each new row of data.

By default, the replicator listens for `GET` requests to `/run` on port `8080`.
This port number can be overridden via the `PORT` environment variable.

## Running

From the test-infra project root, run `make replicator`, then
`bin/replicator -c <config_file>`.

When the replicator receives a `GET` request for `/run`, it will transfer new
data since the last time it was run. When a transfer is in progress, it will
ignore additional requests to `/run` (but still return `200`).

## Requirements and limitations

1. The data in the database must be sequentially ordered by time. Specifically,
   there must be a column of BigQuery datatype TIMESTAMP. This timestamp should
   strictly increase for new data.
2. Configuration only allows for one GCP project at a time.
3. All table names must be unique, even across multiple BigQuery datasets

## How it works

For each table in the configuration, the replicator will search the PostgreSQL
database to determine if that table exists. If it does not exist, the replicator
will attempt to recreate the table in Postgres, based on the BigQuery table's
schema. More on this below.

Once a table in PostgreSQL has been found, the replicator will determine the
timestamp of the most recent row. It will determine this based on the
`dateField` column provided in the configuration.

With the latest timestamp, the replicator will query the BigQuery table for rows
with a newer timestamp. If no timestamp was found (it may be the table was just
created in Postgres), the replicator will copy all data from the associated
BigQuery table.

### Automated table creation and type conversion

For non-nested columns, the replicator currently supports

- FLOAT
- STRING
- TIMESTAMP

BigQuery table fields that are of the `RECORD`/`REPEATED`/`STRUCT`/`ARRAY`
type are converted to JSON and stored in Postgres as the JSON datatype. To
retrieve values from the Postgres table, see the [JSON Functions and Operators]
page.

For example, if your BigQuery table has the FLOAT field `stats.client1.latency`,
this could then be queried and typecast with the following in Postgres: `SELECT
CAST(stats->client1->>'latency' AS DOUBLE PRECISION...`

[JSON Functions and Operators]: https://www.postgresql.org/docs/12/functions-json.html
