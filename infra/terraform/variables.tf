variable "project_id" {
  description = "Google Cloud project ID."
  type        = string
}

variable "region" {
  description = "Primary Google Cloud region."
  type        = string
  default     = "us-central1"
}

variable "zone" {
  description = "Compute Engine zone for VM-based services."
  type        = string
  default     = "us-central1-a"
}

variable "environment" {
  description = "Deployment environment label and application ENVIRONMENT value."
  type        = string
  default     = "production"
}

variable "artifact_repository" {
  description = "Artifact Registry Docker repository name."
  type        = string
  default     = "capstone"
}

variable "image_tag" {
  description = "Container image tag to deploy."
  type        = string
  default     = "latest"
}

variable "frontend_url" {
  description = "Frontend origin. Use the Firebase Hosting default URL after the first frontend deploy."
  type        = string
  default     = "https://CHANGE_ME.web.app"
}

variable "allowed_origins" {
  description = "Comma-separated CORS origins for the API."
  type        = string
  default     = "https://CHANGE_ME.web.app"
}

variable "api_ai_provider" {
  description = "AI provider understood by the API config: anthropic or openai."
  type        = string
  default     = "anthropic"
}

variable "api_ai_model" {
  description = "AI model name passed to the API."
  type        = string
  default     = "claude-opus-4-1"
}

variable "api_ai_api_key" {
  description = "AI provider API key. This is stored in Terraform state if managed here."
  type        = string
  sensitive   = true
}

variable "openai_base_url" {
  description = "Optional OpenAI-compatible base URL, for example an Ollama endpoint."
  type        = string
  default     = ""
}

variable "smtp_host" {
  description = "SMTP host for verification emails."
  type        = string
}

variable "smtp_port" {
  description = "SMTP port for verification emails."
  type        = number
  default     = 587
}

variable "smtp_username" {
  description = "SMTP username."
  type        = string
  sensitive   = true
}

variable "smtp_password" {
  description = "SMTP password. This is stored in Terraform state if managed here."
  type        = string
  sensitive   = true
}

variable "smtp_from" {
  description = "SMTP From header, for example Capstone <no-reply@example.com>."
  type        = string
}

variable "api_min_instances" {
  description = "Minimum API Cloud Run instances."
  type        = number
  default     = 0
}

variable "api_max_instances" {
  description = "Maximum API Cloud Run instances."
  type        = number
  default     = 3
}

variable "control_plane_min_instances" {
  description = "Minimum control-plane Cloud Run instances."
  type        = number
  default     = 1
}

variable "control_plane_max_instances" {
  description = "Maximum control-plane Cloud Run instances."
  type        = number
  default     = 2
}

variable "email_worker_instances" {
  description = "Manual instance count for the Cloud Run email worker pool."
  type        = number
  default     = 1
}

variable "worker_machine_type" {
  description = "Compute Engine machine type for the execution worker."
  type        = string
  default     = "e2-standard-2"
}

variable "worker_artifact_uri" {
  description = "GCS URI for capstone-worker-linux-amd64.tar.gz, created by scripts/build-worker-artifact.sh."
  type        = string
}

variable "worker_languages" {
  description = "Comma-separated language list for the execution worker."
  type        = string
  default     = "python,javascript,go,java"
}

variable "worker_capacity" {
  description = "Maximum concurrent jobs on the execution worker."
  type        = number
  default     = 1
}

variable "worker_docker_runtime" {
  description = "Docker runtime for submitted code containers. Use runsc after installing gVisor, or runc."
  type        = string
  default     = "runc"
}

variable "enable_gvisor" {
  description = "Install gVisor/runsc on the worker VM and use it if worker_docker_runtime is runsc."
  type        = bool
  default     = false
}

variable "cloud_sql_tier" {
  description = "Cloud SQL PostgreSQL machine tier."
  type        = string
  default     = "db-f1-micro"
}

variable "cloud_sql_disk_size_gb" {
  description = "Cloud SQL disk size in GB."
  type        = number
  default     = 20
}
