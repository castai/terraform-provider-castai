package sdk

//go:generate go install github.com/golang/mock/mockgen

//go:generate echo "generating sdk for: ${API_TAGS} from ${SWAGGER_LOCATION}"
//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.4.1 -o api.gen.go --old-config-style -generate types -include-tags $API_TAGS -package sdk $SWAGGER_LOCATION
//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.4.1 -o client.gen.go --old-config-style -templates codegen/templates -generate client -include-tags $API_TAGS -package sdk $SWAGGER_LOCATION
//go:generate mockgen -source client.gen.go -destination mock/client.go . ClientInterface

//go:generate echo "generating sdk for: ${CLUSTER_AUTOSCALER_API_TAGS} from ${CLUSTER_AUTOSCALER_SWAGGER_LOCATION}"
//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.4.1 -o cluster_autoscaler/api.gen.go --old-config-style -generate types -include-tags $CLUSTER_AUTOSCALER_API_TAGS -package cluster_autoscaler $CLUSTER_AUTOSCALER_SWAGGER_LOCATION
//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.4.1 -o cluster_autoscaler/client.gen.go --old-config-style -templates codegen/templates -generate client -include-tags $CLUSTER_AUTOSCALER_API_TAGS -package cluster_autoscaler $CLUSTER_AUTOSCALER_SWAGGER_LOCATION
//go:generate mockgen -source cluster_autoscaler/client.gen.go -destination cluster_autoscaler/mock/client.go . ClientInterface
