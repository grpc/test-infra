module github.com/grpc/test-infra

go 1.16

require (
	cloud.google.com/go/bigquery v1.18.0
	github.com/go-logr/logr v0.1.0
	github.com/golang/protobuf v1.5.2
	github.com/google/uuid v1.2.0
	github.com/jackc/pgx/v4 v4.11.0
	github.com/lib/pq v1.10.2 // indirect
	github.com/onsi/ginkgo v1.12.0
	github.com/onsi/gomega v1.9.0
	github.com/pkg/errors v0.8.1
	golang.org/x/term v0.0.0-20210615171337-6886f2dfbf5b
	golang.org/x/time v0.0.0-20200416051211-89c76fbcd5d1 // indirect
	google.golang.org/api v0.46.0
	google.golang.org/grpc v1.37.0
	k8s.io/api v0.17.2
	k8s.io/apimachinery v0.17.2
	k8s.io/client-go v0.17.2
	sigs.k8s.io/controller-runtime v0.5.0
	sigs.k8s.io/yaml v1.1.0
)
