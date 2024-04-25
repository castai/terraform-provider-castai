package reservations

const (
	FieldReservationsCSV            = "reservations_csv"
	FieldReservationsOrganizationId = "organization_id"
	FieldReservations               = "reservations"
	FieldReservationName            = "name"
	FieldReservationProvider        = "provider"
	FieldReservationRegion          = "region"
	FieldReservationInstanceType    = "instance_type"
	FieldReservationPrice           = "price"
	FieldReservationCount           = "count"
	FieldReservationStartDate       = "start_date"
	FieldReservationEndDate         = "end_date"
	FieldReservationZoneId          = "zone_id"
	FieldReservationZoneName        = "zone_name"

	// Azure specific fields
	FieldReservationProductName           = "product_name"
	FieldReservationQuantity              = "quantity"
	FieldReservationPurchaseDate          = "purchase_date"
	FieldReservationExpirationDate        = "expiration_date"
	FieldReservationType                  = "type"
	FieldReservationDeepLinkToReservation = "deep_link_to_reservation"
	FieldReservationID                    = "reservation_id"
	FieldReservationScopeResourceGroup    = "scope_resource_group"
	FieldReservationScopeSubscription     = "scope_subscription"
	FieldReservationScopeStatus           = "scope_status"
	FieldReservationScope                 = "scope"
	FieldReservationTerm                  = "term"
	FieldReservationStatus                = "status"
)

var csvColumnAlias = map[string][]string{
	FieldReservationName:                  {},
	FieldReservationProvider:              {},
	FieldReservationRegion:                {},
	FieldReservationInstanceType:          {FieldReservationProductName},
	FieldReservationPrice:                 {},
	FieldReservationCount:                 {FieldReservationQuantity},
	FieldReservationStartDate:             {FieldReservationPurchaseDate},
	FieldReservationEndDate:               {FieldReservationExpirationDate},
	FieldReservationZoneId:                {},
	FieldReservationZoneName:              {},
	FieldReservationProductName:           {},
	FieldReservationQuantity:              {},
	FieldReservationPurchaseDate:          {},
	FieldReservationExpirationDate:        {},
	FieldReservationType:                  {},
	FieldReservationDeepLinkToReservation: {},
	FieldReservationID:                    {},
	FieldReservationScopeResourceGroup:    {},
	FieldReservationScopeSubscription:     {},
	FieldReservationScopeStatus:           {},
	FieldReservationTerm:                  {},
	FieldReservationStatus:                {},
	FieldReservationScope:                 {},
}
