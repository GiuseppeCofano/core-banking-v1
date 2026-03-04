output "spanner_instance_name" {
  description = "Name of the created Spanner instance."
  value       = google_spanner_instance.ledger.name
}

output "spanner_database_name" {
  description = "Name of the created Spanner database."
  value       = google_spanner_database.ledger.name
}

output "spanner_database_path" {
  description = "Full resource path to use as the SPANNER_DATABASE env var."
  value       = "projects/${var.project_id}/instances/${google_spanner_instance.ledger.name}/databases/${google_spanner_database.ledger.name}"
}
