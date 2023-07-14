module github.com/pivotal-cf/service-backup

go 1.16

require (
	cloud.google.com/go/storage v1.31.0
	code.cloudfoundry.org/lager v2.0.0+incompatible
	github.com/Azure/azure-sdk-for-go v63.4.0+incompatible
	github.com/Azure/go-autorest/autorest v0.11.18 // indirect
	github.com/Azure/go-autorest/autorest/to v0.4.0 // indirect
	github.com/cenk/backoff v2.2.1+incompatible // indirect
	github.com/craigfurman/herottp v0.0.0-20190418132442-c546d62f2a8d // indirect
	github.com/dnaeon/go-vcr v1.1.0 // indirect
	github.com/gofrs/uuid v4.0.0+incompatible // indirect
	github.com/gorilla/mux v1.8.0 // indirect
	github.com/onsi/ginkgo v1.16.5
	github.com/onsi/gomega v1.19.0
	github.com/pborman/uuid v1.2.1
	github.com/pivotal-cf/service-alerts-client v0.0.0-20190725132148-4a3ed3e6ac41
	github.com/robfig/cron/v3 v3.0.1
	github.com/satori/go.uuid v1.2.0
	github.com/tedsuo/ifrit v0.0.0-20191009134036-9a97d0632f00
	google.golang.org/api v0.130.0
	gopkg.in/yaml.v2 v2.4.0
)
