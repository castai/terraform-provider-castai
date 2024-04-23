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
)

var reservationResourceFields = []string{
	FieldReservationName,
	FieldReservationProvider,
	FieldReservationRegion,
	FieldReservationInstanceType,
	FieldReservationPrice,
	FieldReservationCount,
	FieldReservationStartDate,
	FieldReservationEndDate,
	FieldReservationZoneId,
	FieldReservationZoneName,
	FieldReservationProductName,
	FieldReservationQuantity,
	FieldReservationPurchaseDate,
	FieldReservationExpirationDate,
	FieldReservationType,
	FieldReservationDeepLinkToReservation,
}

var csvColumnAlias = map[string][]string{
	FieldReservationName:                  {FieldReservationName},
	FieldReservationProvider:              {FieldReservationProvider},
	FieldReservationRegion:                {FieldReservationRegion},
	FieldReservationInstanceType:          {FieldReservationInstanceType, FieldReservationProductName},
	FieldReservationPrice:                 {FieldReservationPrice},
	FieldReservationCount:                 {FieldReservationCount, FieldReservationQuantity},
	FieldReservationStartDate:             {FieldReservationStartDate, FieldReservationPurchaseDate},
	FieldReservationEndDate:               {FieldReservationEndDate, FieldReservationExpirationDate},
	FieldReservationZoneId:                {FieldReservationZoneId},
	FieldReservationZoneName:              {FieldReservationZoneName},
	FieldReservationProductName:           {FieldReservationProductName},
	FieldReservationQuantity:              {FieldReservationQuantity},
	FieldReservationPurchaseDate:          {FieldReservationPurchaseDate},
	FieldReservationExpirationDate:        {FieldReservationExpirationDate},
	FieldReservationType:                  {FieldReservationType},
	FieldReservationDeepLinkToReservation: {FieldReservationDeepLinkToReservation},
}
