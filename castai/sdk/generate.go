package sdk

//go:generate echo "generating sdk"

//go:generate go install github.com/deepmap/oapi-codegen/cmd/oapi-codegen@v1.11.0
//go:generate oapi-codegen -config go-sdk.yaml https://api.cast.ai/v1/spec/openapi.json

//go:generate go install github.com/golang/mock/mockgen
//go:generate mockgen -source client.gen.go -destination mock/client.go . ClientWithResponsesInterface
