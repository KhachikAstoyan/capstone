output "artifact_registry_repository" {
  value = google_artifact_registry_repository.docker.name
}

output "api_url" {
  value = google_cloud_run_v2_service.api.uri
}

output "control_plane_url" {
  value = google_cloud_run_v2_service.control_plane.uri
}

output "rabbitmq_internal_ip" {
  value = google_compute_instance.rabbitmq.network_interface[0].network_ip
}

output "worker_instance_name" {
  value = google_compute_instance.worker.name
}
