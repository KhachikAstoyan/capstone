# Google Cloud Deployment

This deployment uses Terraform for infrastructure, Cloud Run for HTTP services,
Firebase Hosting for the Vite frontend, Cloud SQL for PostgreSQL, a private
RabbitMQ VM, and a Compute Engine VM for the execution worker.

## Prerequisites

- A Google Cloud project with billing enabled.
- Local tools: `gcloud`, `terraform`, `go`, `npm`, and optionally `firebase-tools`.
- An SMTP account for verification emails.
- An AI provider API key.

Authenticate once:

```bash
gcloud auth login
gcloud auth application-default login
gcloud config set project <project-id>
```

## Build and Push Container Images

Create the Artifact Registry repository before the first image push. Either
apply only that Terraform target once:

```bash
terraform -chdir=infra/terraform init
terraform -chdir=infra/terraform apply \
  -target=google_project_service.required \
  -target=google_artifact_registry_repository.docker
```

Or create the repository manually with `gcloud`. The Terraform resource uses the
repository name `capstone` by default.

```bash
gcloud builds submit \
  --config cloudbuild.yaml \
  --substitutions _REGION=us-central1,_REPOSITORY=capstone,_TAG=latest
```

This builds and pushes:

- `capstone-api`
- `capstone-control-plane`
- `capstone-email`
- `capstone-python-runner`
- `capstone-js-runner`
- `capstone-go-runner`
- `capstone-java-runner`

## Build and Upload Worker Artifact

The execution worker runs as a Linux binary on Compute Engine because it needs
host Docker access.

```bash
scripts/build-worker-artifact.sh

gcloud storage buckets create gs://<project-id>-capstone-deploy \
  --location=us-central1

gcloud storage cp build/deploy/capstone-worker-linux-amd64.tar.gz \
  gs://<project-id>-capstone-deploy/capstone-worker-linux-amd64.tar.gz
```

Use that `gs://...` path as `worker_artifact_uri` in Terraform.

## Configure Terraform

```bash
cd infra/terraform
cp terraform.tfvars.example terraform.tfvars
```

Edit `terraform.tfvars`:

- `project_id`
- `region` and `zone`
- `worker_artifact_uri`
- `api_ai_api_key`
- `smtp_*`
- temporary `frontend_url` and `allowed_origins`

For the first apply, the frontend URL can be
`https://<project-id>.web.app`. After Firebase deploy, update it to the real URL
shown by Firebase and run `terraform apply` again.

Sensitive values managed by this Terraform are stored in Terraform state. Use a
remote encrypted backend before sharing this outside local development.

## Apply Infrastructure

```bash
terraform init
terraform fmt
terraform validate
terraform plan
terraform apply
```

Terraform creates:

- VPC, subnet, private service access, and Serverless VPC Access connector.
- Artifact Registry repository.
- Cloud SQL PostgreSQL with `capstone` and `capstone_cp`.
- Secret Manager secrets.
- Cloud Run API and internal control plane.
- Cloud Run email worker pool.
- Private RabbitMQ VM.
- Execution worker VM with Docker, runner images, and a systemd service.

## Deploy Frontend

Build the Vite app with the deployed API URL:

```bash
cd web
npm ci
VITE_API_URL="$(terraform -chdir=../infra/terraform output -raw api_url)/api/v1" npm run build
cd ..
```

Deploy to Firebase Hosting:

```bash
npm install -g firebase-tools
firebase login
firebase use --add
firebase deploy --only hosting
```

If Firebase reports a URL different from the placeholder, update
`frontend_url` and `allowed_origins` in `infra/terraform/terraform.tfvars`, then:

```bash
terraform -chdir=infra/terraform apply
```

## Smoke Tests

API:

```bash
API_URL="$(terraform -chdir=infra/terraform output -raw api_url)"
curl "$API_URL/"
```

Control plane from the worker VM:

```bash
gcloud compute ssh capstone-worker-1 --zone <zone> -- \
  'curl -fsS "$WORKER_CP_URL/healthz"'
```

Worker service:

```bash
gcloud compute ssh capstone-worker-1 --zone <zone> -- \
  'sudo systemctl status capstone-worker --no-pager'
```

RabbitMQ:

```bash
gcloud compute ssh capstone-rabbitmq --zone <zone> -- \
  'sudo rabbitmqctl status'
```

## Operational Notes

- The control plane uses internal Cloud Run ingress and still requires
  `X-Internal-Key`.
- RabbitMQ is only reachable from the VPC/subnet and Serverless VPC connector.
- The worker VM pulls Artifact Registry runner images and retags them to the
  local names currently hardcoded by the worker.
- Set `enable_gvisor=true` and `worker_docker_runtime="runsc"` only after
  confirming the gVisor release download succeeds on the VM.
- Rotate local `.env` credentials before production use.
