package sdk

func SupportedClouds() []string {
	return []string{
		string(CloudType_aws),
		string(CloudType_gcp),
		string(CloudType_azure),
		string(CloudType_do),
	}
}
