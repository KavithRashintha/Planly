module github.com/planly/services/gateway

go 1.26.0

require (
	github.com/go-chi/chi/v5 v5.3.2
	github.com/go-chi/cors v1.2.2
	github.com/google/uuid v1.6.0
	github.com/planly/pkg v0.0.0
	github.com/stretchr/testify v1.12.1
	golang.org/x/time v0.16.0
)

require (
	github.com/golang-jwt/jwt/v5 v5.3.1 // indirect
	github.com/kelseyhightower/envconfig v1.4.0 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
)

replace github.com/planly/pkg => ../../pkg
