package sdk

func SupportedClouds() []string {
	return []string{
		string(CloudTypeAws),
		string(CloudTypeGcp),
		string(CloudTypeAzure),
	}
}
