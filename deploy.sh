#!/usr/bin/env bash
# Full GCP deployment for Capstone.
# Usage: bash deploy.sh [--config path/to/deploy.config]
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONFIG_FILE="${SCRIPT_DIR}/deploy.config"

# ─── Parse args ──────────────────────────────────────────────────────────────
while [[ $# -gt 0 ]]; do
  case "$1" in
    --config) CONFIG_FILE="$2"; shift 2 ;;
    *) echo "Unknown arg: $1"; exit 1 ;;
  esac
done

if [[ ! -f "${CONFIG_FILE}" ]]; then
  echo "Config not found: ${CONFIG_FILE}"
  echo "Copy deploy.config.example → deploy.config, fill it in, then re-run."
  exit 1
fi

# shellcheck source=/dev/null
source "${CONFIG_FILE}"

# ─── Validate required vars ──────────────────────────────────────────────────
required_vars=(
  GCP_PROJECT_ID AI_API_KEY
  SMTP_HOST SMTP_USERNAME SMTP_PASSWORD SMTP_FROM
  ADMIN_EMAIL
)
missing=()
for v in "${required_vars[@]}"; do
  [[ -z "${!v:-}" ]] && missing+=("$v")
done
if [[ ${#missing[@]} -gt 0 ]]; then
  echo "Missing required config vars: ${missing[*]}"
  exit 1
fi

# ─── Defaults ────────────────────────────────────────────────────────────────
GCP_REGION="${GCP_REGION:-us-central1}"
GCP_ZONE="${GCP_ZONE:-us-central1-a}"
IMAGE_TAG="${IMAGE_TAG:-latest}"
ARTIFACT_REPOSITORY="${ARTIFACT_REPOSITORY:-capstone}"
AI_PROVIDER="${AI_PROVIDER:-anthropic}"
AI_MODEL="${AI_MODEL:-claude-opus-4-1}"
SMTP_PORT="${SMTP_PORT:-587}"
WORKER_MACHINE_TYPE="${WORKER_MACHINE_TYPE:-e2-standard-2}"
WORKER_LANGUAGES="${WORKER_LANGUAGES:-python,javascript,go,java}"
WORKER_CAPACITY="${WORKER_CAPACITY:-1}"
WORKER_DOCKER_RUNTIME="${WORKER_DOCKER_RUNTIME:-runc}"
ENABLE_GVISOR="${ENABLE_GVISOR:-false}"
CLOUD_SQL_TIER="${CLOUD_SQL_TIER:-db-f1-micro}"
CLOUD_SQL_DISK_GB="${CLOUD_SQL_DISK_GB:-20}"
API_MIN_INSTANCES="${API_MIN_INSTANCES:-0}"
API_MAX_INSTANCES="${API_MAX_INSTANCES:-3}"

BUCKET="${GCP_PROJECT_ID}-worker-artifacts"
ARTIFACT_URI="gs://${BUCKET}/capstone-worker-linux-amd64.tar.gz"
TF_DIR="${SCRIPT_DIR}/infra/terraform"

log() { echo ""; echo "▶ $*"; echo ""; }

# ─── Prerequisites check ─────────────────────────────────────────────────────
log "Checking prerequisites"
for cmd in gcloud terraform go docker; do
  if ! command -v "$cmd" &>/dev/null; then
    echo "Required tool not found: $cmd"
    exit 1
  fi
done

gcloud config set project "${GCP_PROJECT_ID}" --quiet

# ─── Phase 1: Enable APIs + bootstrap Artifact Registry ──────────────────────
log "Enabling GCP APIs"
gcloud services enable \
  artifactregistry.googleapis.com \
  cloudbuild.googleapis.com \
  compute.googleapis.com \
  run.googleapis.com \
  secretmanager.googleapis.com \
  servicenetworking.googleapis.com \
  sqladmin.googleapis.com \
  vpcaccess.googleapis.com \
  storage.googleapis.com \
  --project "${GCP_PROJECT_ID}"

log "Creating Artifact Registry repo (idempotent)"
gcloud artifacts repositories describe "${ARTIFACT_REPOSITORY}" \
  --location "${GCP_REGION}" --project "${GCP_PROJECT_ID}" &>/dev/null || \
gcloud artifacts repositories create "${ARTIFACT_REPOSITORY}" \
  --repository-format docker \
  --location "${GCP_REGION}" \
  --project "${GCP_PROJECT_ID}"

log "Creating GCS bucket for worker artifact (idempotent)"
gcloud storage buckets describe "gs://${BUCKET}" --project "${GCP_PROJECT_ID}" &>/dev/null || \
gcloud storage buckets create "gs://${BUCKET}" \
  --project "${GCP_PROJECT_ID}" \
  --location "${GCP_REGION}"

# ─── Phase 2: Build all Docker images ────────────────────────────────────────
log "Building and pushing Docker images via Cloud Build"
cd "${SCRIPT_DIR}"
gcloud builds submit \
  --config cloudbuild.yaml \
  --substitutions "_REGION=${GCP_REGION},_REPOSITORY=${ARTIFACT_REPOSITORY},_TAG=${IMAGE_TAG}" \
  --project "${GCP_PROJECT_ID}"

# ─── Phase 3: Build worker binary ────────────────────────────────────────────
log "Building worker binary (linux/amd64)"
cd "${SCRIPT_DIR}"
bash scripts/build-worker-artifact.sh

log "Uploading worker binary to GCS"
gcloud storage cp \
  "${SCRIPT_DIR}/build/deploy/capstone-worker-linux-amd64.tar.gz" \
  "${ARTIFACT_URI}"

# ─── Phase 4: Terraform ──────────────────────────────────────────────────────
log "Writing terraform.tfvars"
FRONTEND_URL="${FRONTEND_URL:-http://localhost:5173}"
ALLOWED_ORIGINS="${ALLOWED_ORIGINS:-${FRONTEND_URL}}"

cat > "${TF_DIR}/terraform.tfvars" <<EOF
project_id          = "${GCP_PROJECT_ID}"
region              = "${GCP_REGION}"
zone                = "${GCP_ZONE}"
image_tag           = "${IMAGE_TAG}"
artifact_repository = "${ARTIFACT_REPOSITORY}"

frontend_url        = "${FRONTEND_URL}"
allowed_origins     = "${ALLOWED_ORIGINS}"

api_ai_provider     = "${AI_PROVIDER}"
api_ai_model        = "${AI_MODEL}"
api_ai_api_key      = "${AI_API_KEY}"

smtp_host           = "${SMTP_HOST}"
smtp_port           = ${SMTP_PORT}
smtp_username       = "${SMTP_USERNAME}"
smtp_password       = "${SMTP_PASSWORD}"
smtp_from           = "${SMTP_FROM}"

worker_artifact_uri    = "${ARTIFACT_URI}"
worker_machine_type    = "${WORKER_MACHINE_TYPE}"
worker_languages       = "${WORKER_LANGUAGES}"
worker_capacity        = ${WORKER_CAPACITY}
worker_docker_runtime  = "${WORKER_DOCKER_RUNTIME}"
enable_gvisor          = ${ENABLE_GVISOR}

cloud_sql_tier         = "${CLOUD_SQL_TIER}"
cloud_sql_disk_size_gb = ${CLOUD_SQL_DISK_GB}
api_min_instances      = ${API_MIN_INSTANCES}
api_max_instances      = ${API_MAX_INSTANCES}
EOF

log "Terraform init + apply (this takes ~15 min on first run — Cloud SQL is slow)"
cd "${TF_DIR}"
terraform init -upgrade
terraform apply -auto-approve

API_URL="$(terraform output -raw api_url)"

# ─── Done ─────────────────────────────────────────────────────────────────────
log "Backend deploy complete"
echo ""
echo "  API URL         : ${API_URL}"
echo "  Configured CORS : ${ALLOWED_ORIGINS}"
echo ""
echo "Frontend: build locally and deploy wherever you like (Netlify/Vercel/GH Pages/GCS)."
echo "  cd web"
echo "  echo 'VITE_API_URL=${API_URL}/api/v1' > .env.production"
echo "  npm install && npm run build"
echo "  # dist/ now ready to upload"
echo ""
echo "Local dev frontend: set VITE_API_URL=${API_URL}/api/v1 in web/.env and 'npm run dev'."
echo "  Make sure FRONTEND_URL/ALLOWED_ORIGINS in deploy.config match the URL you serve from."
echo "  Re-run deploy.sh to update CORS if it changes."
echo ""
echo "Assign super admin (after registering an account):"
echo "  cloud-sql-proxy ${GCP_PROJECT_ID}:${GCP_REGION}:capstone-postgres --port 5432 &"
echo "  DB_PASS=\$(gcloud secrets versions access latest --secret=capstone-api-database-url | sed -E 's|.*://capstone:([^@]+)@.*|\\1|')"
echo "  API_DATABASE_URL=\"postgresql://capstone:\$DB_PASS@localhost:5432/capstone?sslmode=disable\" \\"
echo "    bash scripts/assign_super_admin.sh ${ADMIN_EMAIL}"
