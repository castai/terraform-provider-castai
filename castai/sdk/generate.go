package sdk

//go:generate go install github.com/deepmap/oapi-codegen/cmd/oapi-codegen
//go:generate oapi-codegen -o api.gen.go -generate types -package sdk https://api.cast.ai/v1/spec/openapi.json
//go:generate oapi-codegen -o client.gen.go -templates codegen/templates -generate client -package sdk https://api.cast.ai/v1/spec/openapi.json
// //go:generate mockgen -source client.gen.go -destination mock/client.go . ClientInterface
