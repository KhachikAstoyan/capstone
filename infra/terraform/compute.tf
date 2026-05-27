resource "google_compute_instance" "rabbitmq" {
  name         = "${local.name_prefix}-rabbitmq"
  zone         = var.zone
  machine_type = "e2-micro"
  tags         = ["${local.name_prefix}-rabbitmq"]
  labels       = local.common_labels

  boot_disk {
    initialize_params {
      image = "debian-cloud/debian-12"
      size  = 20
      type  = "pd-balanced"
    }
  }

  network_interface {
    subnetwork = google_compute_subnetwork.main.id
  }

  metadata_startup_script = templatefile("${path.module}/scripts/rabbitmq-startup.sh.tftpl", {
    rabbitmq_user     = local.rabbitmq_user
    rabbitmq_password = random_password.rabbitmq_password.result
  })

  depends_on = [google_project_service.required]
}

resource "google_compute_instance" "worker" {
  name         = "${local.name_prefix}-worker-1"
  zone         = var.zone
  machine_type = var.worker_machine_type
  labels       = local.common_labels

  boot_disk {
    initialize_params {
      image = "debian-cloud/debian-12"
      size  = 50
      type  = "pd-balanced"
    }
  }

  network_interface {
    subnetwork = google_compute_subnetwork.main.id
  }

  service_account {
    email  = google_service_account.worker.email
    scopes = ["cloud-platform"]
  }

  metadata_startup_script = templatefile("${path.module}/scripts/worker-startup.sh.tftpl", {
    region                   = var.region
    image_base               = local.image_base
    image_tag                = var.image_tag
    worker_artifact_uri      = var.worker_artifact_uri
    control_plane_key_secret = google_secret_manager_secret.secrets["control-plane-key"].secret_id
    control_plane_url        = google_cloud_run_v2_service.control_plane.uri
    environment              = var.environment
    worker_languages         = var.worker_languages
    worker_capacity          = var.worker_capacity
    worker_docker_runtime    = var.worker_docker_runtime
    enable_gvisor            = tostring(var.enable_gvisor)
  })

  depends_on = [
    google_cloud_run_v2_service.control_plane,
    google_project_iam_member.worker_artifact_reader,
    google_storage_bucket_iam_member.worker_artifact_bucket_viewer,
    google_secret_manager_secret_iam_member.access
  ]
}
