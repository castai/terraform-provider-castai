if [ -z $PROJECT_ID ]; then
	echo "PROJECT_ID environment variable is not defined"
	exit 1
fi

if [ -z $LOCATION ]; then
	echo "LOCATION environment variable is not defined"
	exit 1
fi

if [ -z $CLUSTER_NAME ]; then
	echo "CLUSTER_NAME environment variable is not defined"
	exit 1
fi

SERVICE_ACCOUNTS=($(gcloud container clusters describe $CLUSTER_NAME --zone=$LOCATION --project=$PROJECT_ID --format='value[delimiter="\n"](nodePools[].config.serviceAccount)' | sort | uniq))

PROJECT_NUMBER=$(gcloud projects describe $PROJECT_ID --format="value(projectNumber)")

SERVICE_ACCOUNTS_IDS=()

for sa in "${SERVICE_ACCOUNTS[@]}"; do
	SA=$sa
	if [[ $sa == "default" ]]; then
		SA="${PROJECT_NUMBER}-compute@developer.gserviceaccount.com"
	fi
	ID=$(gcloud iam service-accounts describe $SA --format="json" | jq .uniqueId)
	SERVICE_ACCOUNTS_IDS+=($ID)
done

echo "${SERVICE_ACCOUNTS_IDS[*]}"
