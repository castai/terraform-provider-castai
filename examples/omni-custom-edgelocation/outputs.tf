output "edge_location_id" {
  description = "ID of the created edge location"
  value       = castai_edge_location.this.id
}

output "edge_location_name" {
  description = "Name of the edge location"
  value       = castai_edge_location.this.name
}

output "credentials_revision" {
  description = "Revision number incremented each time credentials change"
  value       = castai_edge_location.this.credentials_revision
}
