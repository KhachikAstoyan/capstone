resource "random_password" "jwt_secret" {
  length  = 48
  special = false
}

resource "random_password" "control_plane_key" {
  length  = 48
  special = false
}

resource "random_password" "rabbitmq_password" {
  length  = 32
  special = false
}

resource "google_secret_manager_secret" "secrets" {
  for_each = toset([
    "api-database-url",
    "cp-database-url",
    "jwt-secret",
    "control-plane-key",
    "ai-api-key",
    "rabbitmq-url",
    "smtp-username",
    "smtp-password"
  ])

  secret_id = "${local.name_prefix}-${each.key}"

  replication {
    auto {}
  }

  depends_on = [google_project_service.required]
}

resource "google_secret_manager_secret_version" "api_database_url" {
  secret      = google_secret_manager_secret.secrets["api-database-url"].id
  secret_data = local.api_database_url
}

resource "google_secret_manager_secret_version" "cp_database_url" {
  secret      = google_secret_manager_secret.secrets["cp-database-url"].id
  secret_data = local.cp_database_url
}

resource "google_secret_manager_secret_version" "jwt_secret" {
  secret      = google_secret_manager_secret.secrets["jwt-secret"].id
  secret_data = random_password.jwt_secret.result
}

resource "google_secret_manager_secret_version" "control_plane_key" {
  secret      = google_secret_manager_secret.secrets["control-plane-key"].id
  secret_data = random_password.control_plane_key.result
}

resource "google_secret_manager_secret_version" "ai_api_key" {
  secret      = google_secret_manager_secret.secrets["ai-api-key"].id
  secret_data = var.api_ai_api_key
}

resource "google_secret_manager_secret_version" "rabbitmq_url" {
  secret      = google_secret_manager_secret.secrets["rabbitmq-url"].id
  secret_data = local.rabbitmq_url
}

resource "google_secret_manager_secret_version" "smtp_username" {
  secret      = google_secret_manager_secret.secrets["smtp-username"].id
  secret_data = var.smtp_username
}

resource "google_secret_manager_secret_version" "smtp_password" {
  secret      = google_secret_manager_secret.secrets["smtp-password"].id
  secret_data = var.smtp_password
}
