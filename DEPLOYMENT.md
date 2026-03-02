# Deployment Guide

## Docker Compose
Use `deploy/docker/docker-compose.yml` with migration/topic one-shot gates.

## Kubernetes
Baseline manifests in `deploy/k8s/`:
- migration Job (`migrate-job.yaml`)
- deployment with readiness/liveness probes (`query-deployment.yaml`)

Use external secret management for `POSTGRES_URL` and key material.
