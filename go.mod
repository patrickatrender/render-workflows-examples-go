module github.com/patrickatrender/render-workflows-examples-go

go 1.26.6

require github.com/render-oss/sdk/go v0.1.0

// pin to a private renderinc/sdk commit; module path there is unchanged from render-oss/sdk
replace github.com/render-oss/sdk/go => github.com/renderinc/sdk/go v0.0.0-20260820023753-3fa63beb7a6f

require (
	github.com/go-chi/chi/v5 v5.3.1 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-retryablehttp v0.7.8 // indirect
	github.com/kelseyhightower/envconfig v1.4.0 // indirect
	github.com/oapi-codegen/runtime v1.2.0 // indirect
)
