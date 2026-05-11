resource "google_service_account" "api" {
  account_id   = "${local.name_prefix}-api"
  display_name = "Capstone API Cloud Run service account"
}

resource "google_service_account" "control_plane" {
  account_id   = "${local.name_prefix}-control-plane"
  display_name = "Capstone control-plane Cloud Run service account"
}

resource "google_service_account" "email" {
  account_id   = "${local.name_prefix}-email"
  display_name = "Capstone email worker service account"
}

resource "google_service_account" "worker" {
  account_id   = "${local.name_prefix}-worker"
  display_name = "Capstone execution worker VM service account"
}

resource "google_project_iam_member" "cloud_sql_client" {
  for_each = {
    api           = google_service_account.api.email
    control_plane = google_service_account.control_plane.email
  }

  project = var.project_id
  role    = "roles/cloudsql.client"
  member  = "serviceAccount:${each.value}"
}

resource "google_project_iam_member" "worker_artifact_reader" {
  project = var.project_id
  role    = "roles/artifactregistry.reader"
  member  = "serviceAccount:${google_service_account.worker.email}"
}

resource "google_project_iam_member" "worker_storage_viewer" {
  project = var.project_id
  role    = "roles/storage.objectViewer"
  member  = "serviceAccount:${google_service_account.worker.email}"
}

locals {
  secret_access = {
    api = {
      service_account = google_service_account.api.email
      secrets = [
        "api-database-url",
        "jwt-secret",
        "control-plane-key",
        "ai-api-key",
        "rabbitmq-url"
      ]
    }
    control_plane = {
      service_account = google_service_account.control_plane.email
      secrets = [
        "cp-database-url",
        "control-plane-key"
      ]
    }
    email = {
      service_account = google_service_account.email.email
      secrets = [
        "rabbitmq-url",
        "smtp-username",
        "smtp-password"
      ]
    }
    worker = {
      service_account = google_service_account.worker.email
      secrets = [
        "control-plane-key"
      ]
    }
  }

  secret_access_pairs = flatten([
    for principal, cfg in local.secret_access : [
      for secret_name in cfg.secrets : {
        key             = "${principal}-${secret_name}"
        service_account = cfg.service_account
        secret_name     = secret_name
      }
    ]
  ])
}

resource "google_secret_manager_secret_iam_member" "access" {
  for_each = {
    for pair in local.secret_access_pairs : pair.key => pair
  }

  secret_id = google_secret_manager_secret.secrets[each.value.secret_name].id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${each.value.service_account}"
}
