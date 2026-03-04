terraform {
  required_version = ">= 1.5"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 5.0"
    }
  }

  # Uncomment and configure for remote state:
  # backend "gcs" {
  #   bucket = "your-terraform-state-bucket"
  #   prefix = "core-banking/spanner"
  # }
}

provider "google" {
  project = var.project_id
  region  = var.region
}

# ──────────────────────────────────────────────────────────────────────────────
# Spanner Instance
# ──────────────────────────────────────────────────────────────────────────────
resource "google_spanner_instance" "ledger" {
  name             = var.spanner_instance_name
  config           = var.spanner_config
  display_name     = "Core Banking Ledger"
  processing_units = var.processing_units

  labels = {
    environment = var.environment
    service     = "ledger"
    managed-by  = "terraform"
  }
}

# ──────────────────────────────────────────────────────────────────────────────
# Spanner Database with DDL
# ──────────────────────────────────────────────────────────────────────────────
resource "google_spanner_database" "ledger" {
  instance = google_spanner_instance.ledger.name
  name     = var.spanner_database_name

  # Prevent Terraform from destroying the database (data loss protection).
  deletion_protection = var.deletion_protection

  ddl = [
    <<-SQL
      CREATE TABLE accounts (
        id         STRING(36)  NOT NULL,
        owner      STRING(256) NOT NULL,
        currency   STRING(3)   NOT NULL,
        balance    FLOAT64     NOT NULL,
        created_at TIMESTAMP   NOT NULL,
        updated_at TIMESTAMP   NOT NULL,
      ) PRIMARY KEY (id)
    SQL
    ,
    <<-SQL
      CREATE TABLE ledger_entries (
        account_id     STRING(36)  NOT NULL,
        id             STRING(36)  NOT NULL,
        transaction_id STRING(36)  NOT NULL,
        type           STRING(32)  NOT NULL,
        amount         FLOAT64     NOT NULL,
        balance        FLOAT64     NOT NULL,
        description    STRING(1024),
        created_at     TIMESTAMP   NOT NULL,
      ) PRIMARY KEY (account_id, id),
        INTERLEAVE IN PARENT accounts ON DELETE CASCADE
    SQL
    ,
    "CREATE INDEX idx_ledger_entries_transaction ON ledger_entries(transaction_id)",
  ]
}

# ──────────────────────────────────────────────────────────────────────────────
# IAM – Grant the GKE workload identity access to Spanner
# ──────────────────────────────────────────────────────────────────────────────
resource "google_spanner_database_iam_member" "ledger_sa" {
  count    = var.ledger_service_account != "" ? 1 : 0
  instance = google_spanner_instance.ledger.name
  database = google_spanner_database.ledger.name
  role     = "roles/spanner.databaseUser"
  member   = "serviceAccount:${var.ledger_service_account}"
}
