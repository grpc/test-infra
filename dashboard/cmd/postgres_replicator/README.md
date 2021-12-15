# Postgres replicator

This tool replicates streaming data from BigQuery into PostgreSQL.


## Configuration

It is configured with a YAML file. Here is an example configuration file:

```
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

## Limitations

- Configuration only allows for one GCP project at a time.
- All table names must be unique, even across multiple BigQuery datasets
