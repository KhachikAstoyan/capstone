resource "random_password" "db_password" {
  length  = 32
  special = false
}

resource "google_sql_database_instance" "postgres" {
  name             = "${local.name_prefix}-postgres"
  database_version = "POSTGRES_17"
  region           = var.region

  settings {
    tier              = var.cloud_sql_tier
    availability_type = "ZONAL"
    disk_size         = var.cloud_sql_disk_size_gb
    disk_type         = "PD_SSD"

    ip_configuration {
      ipv4_enabled    = false
      private_network = google_compute_network.main.id
    }

    backup_configuration {
      enabled                        = true
      point_in_time_recovery_enabled = true
    }
  }

  deletion_protection = true

  depends_on = [
    google_project_service.required,
    google_service_networking_connection.private_vpc_connection
  ]
}

resource "google_sql_user" "app" {
  name     = local.db_user
  instance = google_sql_database_instance.postgres.name
  password = random_password.db_password.result
}

resource "google_sql_database" "api" {
  name     = "capstone"
  instance = google_sql_database_instance.postgres.name
}

resource "google_sql_database" "control_plane" {
  name     = "capstone_cp"
  instance = google_sql_database_instance.postgres.name
}
