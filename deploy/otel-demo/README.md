# OTel Demo — Kubilitics Feature Testing

A lightweight multi-service demo that generates real OpenTelemetry traces
for testing Kubilitics tracing features. No external dependencies required.

## Quick Start

```bash
# Deploy everything (namespace, collector, 3 services, load generator)
kubectl apply -f deploy/otel-demo/

# Watch pods come up
kubectl get pods -n otel-demo -w

# Check traces are flowing (after ~30s)
curl http://localhost:8190/api/v1/clusters/<cluster-id>/traces
```

## Architecture

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│ order-service │────▶│ payment-svc  │────▶│ inventory-svc│
│    :8080      │     │    :8080     │     │    :8080     │
└──────┬───────┘     └──────┬───────┘     └──────┬───────┘
       │                    │                    │
       └────────────┬───────┘────────────────────┘
                    ▼
           ┌───────────────┐
           │ OTel Collector │
           │  :4318 (HTTP)  │
           └───────┬───────┘
                   ▼
           ┌───────────────┐
           │  Kubilitics    │
           │  Backend       │
           │  :8190/v1/otel │
           └───────────────┘
```

## What It Generates

- **order-service**: HTTP server, creates orders, calls payment + inventory
- **payment-service**: Processes payments, simulates Stripe API calls
- **inventory-service**: Checks stock, simulates DB queries
- **load-generator**: Sends requests every 2s to order-service

Each service has:
- 10% error rate (realistic)
- Variable latency (10-200ms)
- DB and cache span simulation
- Proper OTel resource attributes (service name, namespace, pod)

## Cleanup

```bash
kubectl delete namespace otel-demo
```
