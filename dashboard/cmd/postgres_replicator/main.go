package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	pgr "github.com/grpc/test-infra/dashboard/postgres_replicator"
	_ "github.com/jackc/pgx/v4/stdlib"
)

func main() {
	var c string
	flag.StringVar(&c, "c", "", "filepath to config")
	flag.Parse()

	if c == "" {
		fmt.Fprintf(os.Stderr, "Usage: postgres_replicator -c <config>\n")
		os.Exit(1)
	}

	config, err := pgr.NewConfig(c)
	if err != nil {
		log.Fatalf("Error getting config: %s", err)
	}

	var (
		postgresConfig = config.Postgres
		bigqueryConfig = config.BigQuery
		transferConfig = config.Transfer
	)

	pgdb, err := pgr.NewPostgresClient(postgresConfig)
	if err != nil {
		log.Fatalf("Error initializing PostgreSQL client: %v", err)
	}
	log.Println("Initialized PostgreSQL client")

	bqdb, err := pgr.NewBigQueryClient(context.Background(), bigqueryConfig)
	if err != nil {
		log.Fatalf("Error initializing BigQuery client: %v", err)
	}
	log.Println("Initialized BigQuery client")

	dbTransfer := pgr.NewTransfer(bqdb, pgdb, &transferConfig)
	finished := make(chan bool)
	go serveHTTP(dbTransfer, finished)

	<-finished
}

func serveHTTP(dbTransfer *pgr.Transfer, finished chan bool) {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "Alive")
	})
	http.HandleFunc("/run", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Request received")
		go dbTransfer.Run()
	})
	http.HandleFunc("/kill", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Server killed")
		finished <- true
	})

	log.Printf("Listening on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
