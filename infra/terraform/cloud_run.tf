resource "google_cloud_run_v2_service" "control_plane" {
  name                = "${local.name_prefix}-control-plane"
  location            = var.region
  ingress             = "INGRESS_TRAFFIC_INTERNAL_ONLY"
  labels              = local.common_labels
  deletion_protection = false

  template {
    service_account = google_service_account.control_plane.email

    scaling {
      min_instance_count = var.control_plane_min_instances
      max_instance_count = var.control_plane_max_instances
    }

    vpc_access {
      network_interfaces {
        network    = google_compute_network.main.id
        subnetwork = google_compute_subnetwork.main.id
      }
      egress = "PRIVATE_RANGES_ONLY"
    }

    volumes {
      name = "cloudsql"
      cloud_sql_instance {
        instances = [google_sql_database_instance.postgres.connection_name]
      }
    }

    containers {
      image = local.control_plane_image

      ports {
        container_port = 9090
      }

      env {
        name  = "ENVIRONMENT"
        value = var.environment
      }
      env {
        name  = "CP_PORT"
        value = "9090"
      }
      env {
        name = "CP_DATABASE_URL"
        value_source {
          secret_key_ref {
            secret  = google_secret_manager_secret.secrets["cp-database-url"].secret_id
            version = "latest"
          }
        }
      }
      env {
        name = "CP_INTERNAL_KEY"
        value_source {
          secret_key_ref {
            secret  = google_secret_manager_secret.secrets["control-plane-key"].secret_id
            version = "latest"
          }
        }
      }
      env {
        name = "CP_RABBITMQ_URL"
        value_source {
          secret_key_ref {
            secret  = google_secret_manager_secret.secrets["rabbitmq-url"].secret_id
            version = "latest"
          }
        }
      }

      volume_mounts {
        name       = "cloudsql"
        mount_path = "/cloudsql"
      }

      startup_probe {
        http_get {
          path = "/healthz"
          port = 9090
        }
      }

      resources {
        limits = {
          cpu    = "1"
          memory = "512Mi"
        }
      }
    }
  }

  traffic {
    percent = 100
    type    = "TRAFFIC_TARGET_ALLOCATION_TYPE_LATEST"
  }

  depends_on = [
    google_project_iam_member.cloud_sql_client,
    google_secret_manager_secret_iam_member.access,
    google_secret_manager_secret_version.cp_database_url,
    google_secret_manager_secret_version.control_plane_key,
    google_secret_manager_secret_version.rabbitmq_url
  ]
}

resource "google_cloud_run_v2_service" "api" {
  name                = "${local.name_prefix}-api"
  location            = var.region
  ingress             = "INGRESS_TRAFFIC_ALL"
  labels              = local.common_labels
  deletion_protection = false

  template {
    service_account = google_service_account.api.email

    scaling {
      min_instance_count = var.api_min_instances
      max_instance_count = var.api_max_instances
    }

    vpc_access {
      network_interfaces {
        network    = google_compute_network.main.id
        subnetwork = google_compute_subnetwork.main.id
      }
      egress = "PRIVATE_RANGES_ONLY"
    }

    volumes {
      name = "cloudsql"
      cloud_sql_instance {
        instances = [google_sql_database_instance.postgres.connection_name]
      }
    }

    containers {
      image = local.api_image

      ports {
        container_port = 8080
      }

      env {
        name  = "ENVIRONMENT"
        value = var.environment
      }
      env {
        name  = "API_PORT"
        value = "8080"
      }
      env {
        name  = "API_SECURE_COOKIES"
        value = "true"
      }
      env {
        name  = "API_ALLOWED_ORIGINS"
        value = var.allowed_origins
      }
      env {
        name  = "API_FRONTEND_URL"
        value = var.frontend_url
      }
      env {
        name  = "API_CONTROL_PLANE_URL"
        value = google_cloud_run_v2_service.control_plane.uri
      }
      env {
        name  = "AI_PROVIDER"
        value = var.api_ai_provider
      }
      env {
        name  = "AI_MODEL"
        value = var.api_ai_model
      }
      dynamic "env" {
        for_each = var.openai_base_url != "" ? [1] : []
        content {
          name  = "OPENAI_BASE_URL"
          value = var.openai_base_url
        }
      }
      env {
        name = "API_DATABASE_URL"
        value_source {
          secret_key_ref {
            secret  = google_secret_manager_secret.secrets["api-database-url"].secret_id
            version = "latest"
          }
        }
      }
      env {
        name = "JWT_SECRET"
        value_source {
          secret_key_ref {
            secret  = google_secret_manager_secret.secrets["jwt-secret"].secret_id
            version = "latest"
          }
        }
      }
      env {
        name = "API_CONTROL_PLANE_KEY"
        value_source {
          secret_key_ref {
            secret  = google_secret_manager_secret.secrets["control-plane-key"].secret_id
            version = "latest"
          }
        }
      }
      env {
        name = "AI_API_KEY"
        value_source {
          secret_key_ref {
            secret  = google_secret_manager_secret.secrets["ai-api-key"].secret_id
            version = "latest"
          }
        }
      }
      env {
        name = "ANTHROPIC_API_KEY"
        value_source {
          secret_key_ref {
            secret  = google_secret_manager_secret.secrets["ai-api-key"].secret_id
            version = "latest"
          }
        }
      }
      env {
        name = "OPENAI_API_KEY"
        value_source {
          secret_key_ref {
            secret  = google_secret_manager_secret.secrets["ai-api-key"].secret_id
            version = "latest"
          }
        }
      }
      env {
        name = "API_RABBITMQ_URL"
        value_source {
          secret_key_ref {
            secret  = google_secret_manager_secret.secrets["rabbitmq-url"].secret_id
            version = "latest"
          }
        }
      }

      volume_mounts {
        name       = "cloudsql"
        mount_path = "/cloudsql"
      }

      startup_probe {
        http_get {
          path = "/"
          port = 8080
        }
      }

      resources {
        limits = {
          cpu    = "1"
          memory = "512Mi"
        }
      }
    }
  }

  traffic {
    percent = 100
    type    = "TRAFFIC_TARGET_ALLOCATION_TYPE_LATEST"
  }

  depends_on = [
    google_cloud_run_v2_service.control_plane,
    google_project_iam_member.cloud_sql_client,
    google_secret_manager_secret_iam_member.access,
    google_secret_manager_secret_version.api_database_url,
    google_secret_manager_secret_version.jwt_secret,
    google_secret_manager_secret_version.control_plane_key,
    google_secret_manager_secret_version.ai_api_key,
    google_secret_manager_secret_version.rabbitmq_url
  ]
}

resource "google_cloud_run_v2_service_iam_member" "api_public_invoker" {
  location = google_cloud_run_v2_service.api.location
  name     = google_cloud_run_v2_service.api.name
  role     = "roles/run.invoker"
  member   = "allUsers"
}

resource "google_cloud_run_v2_worker_pool" "email" {
  provider = google-beta

  name                = "${local.name_prefix}-email"
  location            = var.region
  launch_stage        = "BETA"
  labels              = local.common_labels
  deletion_protection = false

  template {
    service_account = google_service_account.email.email

    vpc_access {
      network_interfaces {
        network    = google_compute_network.main.id
        subnetwork = google_compute_subnetwork.main.id
      }
      egress = "PRIVATE_RANGES_ONLY"
    }

    containers {
      image = local.email_image

      env {
        name  = "ENVIRONMENT"
        value = var.environment
      }
      env {
        name  = "SMTP_HOST"
        value = var.smtp_host
      }
      env {
        name  = "SMTP_PORT"
        value = tostring(var.smtp_port)
      }
      env {
        name  = "SMTP_FROM"
        value = var.smtp_from
      }
      env {
        name = "EMAIL_RABBITMQ_URL"
        value_source {
          secret_key_ref {
            secret  = google_secret_manager_secret.secrets["rabbitmq-url"].secret_id
            version = "latest"
          }
        }
      }
      env {
        name = "SMTP_USERNAME"
        value_source {
          secret_key_ref {
            secret  = google_secret_manager_secret.secrets["smtp-username"].secret_id
            version = "latest"
          }
        }
      }
      env {
        name = "SMTP_PASSWORD"
        value_source {
          secret_key_ref {
            secret  = google_secret_manager_secret.secrets["smtp-password"].secret_id
            version = "latest"
          }
        }
      }

      resources {
        limits = {
          cpu    = "1"
          memory = "512Mi"
        }
      }
    }
  }

  scaling {
    scaling_mode          = "MANUAL"
    manual_instance_count = var.email_worker_instances
  }

  depends_on = [
    google_secret_manager_secret_iam_member.access,
    google_secret_manager_secret_version.rabbitmq_url,
    google_secret_manager_secret_version.smtp_username,
    google_secret_manager_secret_version.smtp_password
  ]
}
