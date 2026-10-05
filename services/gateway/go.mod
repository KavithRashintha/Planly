module github.com/planly/services/gateway

go 1.23

require (
	github.com/go-chi/chi/v5 v5.3.2
	github.com/planly/pkg v0.0.0
)

require (
	github.com/google/uuid v1.6.0 // indirect
	github.com/kelseyhightower/envconfig v1.4.0 // indirect
)

replace github.com/planly/pkg => ../../pkg
