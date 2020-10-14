package sdk

//go:generate go install github.com/deepmap/oapi-codegen/cmd/oapi-codegen
//go:generate oapi-codegen -o api.gen.go -generate types -package sdk spec.yaml
//go:generate oapi-codegen -o client.gen.go -templates codegen/templates -generate client -package sdk spec.yaml
