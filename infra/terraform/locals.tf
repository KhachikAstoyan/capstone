locals {
  name_prefix = "capstone"

  image_base = "${var.region}-docker.pkg.dev/${var.project_id}/${var.artifact_repository}"

  api_image           = "${local.image_base}/capstone-api:${var.image_tag}"
  control_plane_image = "${local.image_base}/capstone-control-plane:${var.image_tag}"
  email_image         = "${local.image_base}/capstone-email:${var.image_tag}"

  db_user = "capstone"

  api_database_url = "postgresql://${local.db_user}:${random_password.db_password.result}@/${google_sql_database.api.name}?host=/cloudsql/${google_sql_database_instance.postgres.connection_name}&sslmode=disable"
  cp_database_url  = "postgresql://${local.db_user}:${random_password.db_password.result}@/${google_sql_database.control_plane.name}?host=/cloudsql/${google_sql_database_instance.postgres.connection_name}&sslmode=disable"
  rabbitmq_url     = "amqp://${local.rabbitmq_user}:${random_password.rabbitmq_password.result}@${google_compute_instance.rabbitmq.network_interface[0].network_ip}:5672/"
  rabbitmq_user    = "capstone"

  common_labels = {
    app         = "capstone"
    environment = var.environment
    managed_by  = "terraform"
  }
}
