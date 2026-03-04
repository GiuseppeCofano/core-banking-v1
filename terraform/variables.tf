variable "project_id" {
  description = "GCP project ID where the Spanner instance will be created."
  type        = string
}

variable "region" {
  description = "GCP region for the provider default (not the Spanner config)."
  type        = string
  default     = "us-central1"
}

variable "spanner_instance_name" {
  description = "Name of the Spanner instance."
  type        = string
  default     = "core-banking-ledger"
}

variable "spanner_database_name" {
  description = "Name of the Spanner database."
  type        = string
  default     = "ledger"
}

variable "spanner_config" {
  description = "Spanner instance configuration (e.g. regional-us-central1, nam-eur-asia1)."
  type        = string
  default     = "regional-us-central1"
}

variable "processing_units" {
  description = "Processing units for the Spanner instance (100 = smallest, 1000 = 1 node)."
  type        = number
  default     = 100
}

variable "environment" {
  description = "Environment label (e.g. dev, staging, production)."
  type        = string
  default     = "dev"
}

variable "deletion_protection" {
  description = "Prevent Terraform from deleting the database."
  type        = bool
  default     = true
}

variable "ledger_service_account" {
  description = "GCP service account email for the ledger workload (e.g. for GKE Workload Identity). Leave empty to skip IAM binding."
  type        = string
  default     = ""
}
