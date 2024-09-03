package sdk

//go:generate echo "generating sdk for: ${API_TAGS} from ${SWAGGER_LOCATION}"
//go:generate go install github.com/deepmap/oapi-codegen/cmd/oapi-codegen@v1.11.0
//go:generate oapi-codegen -o api.gen.go --old-config-style -generate types -include-tags $API_TAGS -package sdk $SWAGGER_LOCATION
//go:generate oapi-codegen -o client.gen.go --old-config-style -templates codegen/templates -generate client -include-tags $API_TAGS -package sdk $SWAGGER_LOCATION
//go:generate go install github.com/golang/mock/mockgen
//go:generate mockgen -source client.gen.go -destination mock/client.go . ClientInterface
