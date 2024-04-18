package commitments

const (
	FieldAzureReservationsCSV = "azure_reservations_csv"
	FieldGCPCUDsJSON          = "gcp_cuds_json"

	FieldAzureReservations = "azure_reservations"
	FieldGCPCUDs           = "gcp_cuds"

	// Common fields

	FieldName     = "name"
	FieldRegion   = "region"
	FieldStatus   = "status"
	FieldType     = "type"
	FieldCPU      = "cpu"
	FieldMemoryMB = "memory_mb"

	// GCP CUD fields

	FieldAutoRenew         = "auto_renew"
	FieldCategory          = "category"
	FieldCreationTimestamp = "creation_timestamp"
	FieldDescription       = "description"
	FieldEndTimestamp      = "end_timestamp"
	FieldId                = "id"
	FieldKind              = "kind"
	FieldPlan              = "plan"
	FieldResources         = "resources"
	FieldSelfLink          = "self_link"
	FieldStartTimestamp    = "start_timestamp"
	FieldStatusMessage     = "status_message"
	FieldAmount            = "amount"

	// Azure Reservation fields

	FieldExpirationDate     = "expiration_date"
	FieldProductName        = "product_name"
	FieldPurchaseDate       = "purchase_date"
	FieldQuantity           = "quantity"
	FieldReservationId      = "reservation_id"
	FieldScope              = "scope"
	FieldScopeResourceGroup = "scope_resource_group"
	FieldScopeSubscription  = "scope_subscription"
	FieldTerm               = "term"
)
