# Cloud Build Specification

## Realtime Gateway — `backend/realtime-gateway/cloudbuild.yaml`

```yaml
steps:
  - name: 'golang:1.24'
    args: ['go', 'test', './...']
    dir: 'backend/realtime-gateway'

  - name: 'gcr.io/cloud-builders/docker'
    args: ['build', '-t', 'asia-northeast3-docker.pkg.dev/vibecat-489105/vibecat/realtime-gateway:$COMMIT_SHA', '.']
    dir: 'backend/realtime-gateway'

  - name: 'gcr.io/cloud-builders/docker'
    args: ['push', 'asia-northeast3-docker.pkg.dev/vibecat-489105/vibecat/realtime-gateway:$COMMIT_SHA']

  - name: 'gcr.io/google.com/cloudsdktool/cloud-sdk'
    entrypoint: 'gcloud'
    args: ['run', 'deploy', 'realtime-gateway',
           '--image', 'asia-northeast3-docker.pkg.dev/vibecat-489105/vibecat/realtime-gateway:$COMMIT_SHA',
           '--region', 'asia-northeast3',
           '--port', '8080',
           '--memory', '512Mi',
           '--min-instances', '0',
           '--max-instances', '10',
           '--concurrency', '80']

images:
  - 'asia-northeast3-docker.pkg.dev/vibecat-489105/vibecat/realtime-gateway:$COMMIT_SHA'
```

## ADK Orchestrator — `backend/adk-orchestrator/cloudbuild.yaml`

```yaml
steps:
  - name: 'golang:1.24'
    args: ['go', 'test', './...']
    dir: 'backend/adk-orchestrator'

  - name: 'gcr.io/cloud-builders/docker'
    args: ['build', '-t', 'asia-northeast3-docker.pkg.dev/vibecat-489105/vibecat/adk-orchestrator:$COMMIT_SHA', '.']
    dir: 'backend/adk-orchestrator'

  - name: 'gcr.io/cloud-builders/docker'
    args: ['push', 'asia-northeast3-docker.pkg.dev/vibecat-489105/vibecat/adk-orchestrator:$COMMIT_SHA']

  - name: 'gcr.io/google.com/cloudsdktool/cloud-sdk'
    entrypoint: 'gcloud'
    args: ['run', 'deploy', 'adk-orchestrator',
           '--image', 'asia-northeast3-docker.pkg.dev/vibecat-489105/vibecat/adk-orchestrator:$COMMIT_SHA',
           '--region', 'asia-northeast3',
           '--port', '8080',
           '--memory', '1Gi',
           '--min-instances', '0',
           '--max-instances', '10',
           '--concurrency', '100']

images:
  - 'asia-northeast3-docker.pkg.dev/vibecat-489105/vibecat/adk-orchestrator:$COMMIT_SHA'
```

## Artifact Registry
- Repository: `asia-northeast3-docker.pkg.dev/vibecat-489105/vibecat`
- Type: Docker
- Region: `asia-northeast3`

## Build Triggers
- Push to `master` → build + deploy both services
