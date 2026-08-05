module github.com/castai/terraform-provider-castai

go 1.25.5

require (
	github.com/cenkalti/backoff/v4 v4.1.3
	github.com/evanphx/json-patch v4.12.0+incompatible
	github.com/golang/mock v1.6.0
	github.com/google/uuid v1.6.0
	github.com/gruntwork-io/terratest v0.40.18
	github.com/hashicorp/go-cty v1.5.0
	github.com/hashicorp/terraform-plugin-framework v1.19.0
	github.com/hashicorp/terraform-plugin-framework-validators v0.19.0
	github.com/hashicorp/terraform-plugin-go v0.31.0
	github.com/hashicorp/terraform-plugin-log v0.10.0
	github.com/hashicorp/terraform-plugin-mux v0.23.0
	github.com/hashicorp/terraform-plugin-sdk/v2 v2.40.0
	github.com/hashicorp/terraform-plugin-testing v1.15.0
	github.com/joho/godotenv v1.4.0
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/mitchellh/mapstructure v1.5.0
	github.com/oapi-codegen/runtime v1.1.1
	github.com/samber/lo v1.49.1
	github.com/stretchr/testify v1.11.1
	golang.org/x/crypto v0.54.0
	golang.org/x/exp v0.0.0-20230626212559-97b1e661b5df
	k8s.io/apimachinery v0.30.0
)

require (
	cel.dev/expr v0.25.1 // indirect
	cloud.google.com/go v0.121.4 // indirect
	cloud.google.com/go/auth v0.17.0 // indirect
	cloud.google.com/go/auth/oauth2adapt v0.2.8 // indirect
	cloud.google.com/go/compute/metadata v0.9.0 // indirect
	cloud.google.com/go/iam v1.5.2 // indirect
	cloud.google.com/go/monitoring v1.24.2 // indirect
	cloud.google.com/go/storage v1.56.0 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/detectors/gcp v1.32.0 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/metric v0.53.0 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/internal/resourcemapping v0.53.0 // indirect
	github.com/ProtonMail/go-crypto v1.3.0 // indirect
	github.com/agext/levenshtein v1.2.3 // indirect
	github.com/apapsch/go-jsonmerge/v2 v2.0.0 // indirect
	github.com/apparentlymart/go-textseg/v15 v15.0.0 // indirect
	github.com/aws/aws-sdk-go v1.44.122 // indirect
	github.com/bgentry/go-netrc v0.0.0-20140422174119-9fd32a8b3d3d // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cloudflare/circl v1.6.1 // indirect
	github.com/cncf/xds/go v0.0.0-20260202195803-dba9d589def2 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/envoyproxy/go-control-plane/envoy v1.37.0 // indirect
	github.com/envoyproxy/protoc-gen-validate v1.3.3 // indirect
	github.com/fatih/color v1.18.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/go-jose/go-jose/v4 v4.1.4 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/google/s2a-go v0.1.9 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.7 // indirect
	github.com/googleapis/gax-go/v2 v2.15.0 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-checkpoint v0.5.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-getter v1.7.9 // indirect
	github.com/hashicorp/go-hclog v1.6.3 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/go-plugin v1.7.0 // indirect
	github.com/hashicorp/go-retryablehttp v0.7.8 // indirect
	github.com/hashicorp/go-safetemp v1.0.0 // indirect
	github.com/hashicorp/go-uuid v1.0.3 // indirect
	github.com/hashicorp/go-version v1.8.0 // indirect
	github.com/hashicorp/hc-install v0.9.3 // indirect
	github.com/hashicorp/hcl/v2 v2.24.0 // indirect
	github.com/hashicorp/logutils v1.0.0 // indirect
	github.com/hashicorp/terraform-exec v0.25.0 // indirect
	github.com/hashicorp/terraform-json v0.27.2 // indirect
	github.com/hashicorp/terraform-registry-address v0.4.0 // indirect
	github.com/hashicorp/terraform-svchost v0.1.1 // indirect
	github.com/hashicorp/yamux v0.1.2 // indirect
	github.com/jinzhu/copier v0.0.0-20190924061706-b57f9002281a // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/klauspost/compress v1.16.7 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-zglob v0.0.2-0.20190814121620-e3c945676326 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/go-testing-interface v1.14.1 // indirect
	github.com/mitchellh/go-wordwrap v1.0.1 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/oklog/run v1.1.0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/planetscale/vtprotobuf v0.6.1-0.20240319094008-0393e58bdf10 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/spiffe/go-spiffe/v2 v2.6.0 // indirect
	github.com/tmccombs/hcl2json v0.3.3 // indirect
	github.com/ulikunitz/xz v0.5.10 // indirect
	github.com/vmihailenco/msgpack v4.0.4+incompatible // indirect
	github.com/vmihailenco/msgpack/v5 v5.4.1 // indirect
	github.com/vmihailenco/tagparser/v2 v2.0.0 // indirect
	github.com/zclconf/go-cty v1.17.0 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/contrib/detectors/gcp v1.43.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.61.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.61.0 // indirect
	go.opentelemetry.io/otel v1.43.0 // indirect
	go.opentelemetry.io/otel/metric v1.43.0 // indirect
	go.opentelemetry.io/otel/sdk v1.43.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.43.0 // indirect
	go.opentelemetry.io/otel/trace v1.43.0 // indirect
	golang.org/x/mod v0.37.0 // indirect
	golang.org/x/net v0.56.0 // indirect
	golang.org/x/oauth2 v0.36.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.40.0 // indirect
	golang.org/x/time v0.14.0 // indirect
	golang.org/x/tools v0.47.0 // indirect
	google.golang.org/api v0.256.0 // indirect
	google.golang.org/appengine v1.6.8 // indirect
	google.golang.org/genproto v0.0.0-20250603155806-513f23925822 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20260414002931-afd174a4e478 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260414002931-afd174a4e478 // indirect
	google.golang.org/grpc v1.82.1 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	k8s.io/klog/v2 v2.130.1 // indirect
	k8s.io/utils v0.0.0-20251002143259-bc988d571ff4 // indirect
)

exclude github.com/satori/go.uuid v1.2.0

// go-getter pulls mock v1.6, which makes DoAndReturn enforce variadic RequestEditorFn arity and breaks existing unit tests.
replace github.com/golang/mock => github.com/golang/mock v1.5.0
