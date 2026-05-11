resource "google_artifact_registry_repository" "docker" {
  location      = var.region
  repository_id = var.artifact_repository
  description   = "Capstone Docker images"
  format        = "DOCKER"

  depends_on = [google_project_service.required]
}
