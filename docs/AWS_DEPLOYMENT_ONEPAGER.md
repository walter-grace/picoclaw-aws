# Claw Cubed on AWS: Deployment One-Pager

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                         User / Chat Client                        │
└─────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────┐
│  API Gateway (HTTP API)  │  WebSocket API  │  Telegram/Discord     │
└─────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Lambda (Claw Cubed runtime)                    │
│              or ECS Fargate / App Runner (long-lived)             │
└─────────────────────────────────────────────────────────────────┘
                                    │
                    ┌───────────────┼───────────────┐
                    ▼               ▼               ▼
            ┌───────────┐   ┌───────────┐   ┌───────────┐
            │    S3     │   │  Secrets  │   │  Bedrock   │
            │  Memory   │   │  Manager  │   │  (optional)│
            └───────────┘   └───────────┘   └───────────┘
```

---

## AWS Services You Need

| Service | Purpose |
|---------|---------|
| **S3** | Memory backend (`MEMORY.md`, daily notes). One bucket per deployment or per tenant. |
| **Lambda** | Run Claw Cubed as serverless (stateless handler, memory in S3). |
| **ECS Fargate** or **App Runner** | Run as long-lived container if you need WebSockets or persistent connections. |
| **API Gateway** | HTTP API for REST, or WebSocket API for real-time chat. |
| **Secrets Manager** | Store LLM API keys, Telegram/Discord tokens, AWS credentials. |
| **IAM** | Roles for Lambda/ECS to access S3, Secrets Manager, Bedrock. |
| **ECR** | Store Docker image of Claw Cubed. |
| **CloudWatch** | Logs and metrics. |

---

## Deployment Options

### Option A: Lambda + API Gateway (Serverless)

1. Build Claw Cubed as a Go binary or container.
2. Package as Lambda (custom runtime or container image).
3. Create HTTP API or WebSocket API in API Gateway.
4. Lambda reads/writes memory in S3; no local disk.
5. **Pros:** Pay per request, auto-scaling. **Cons:** Cold starts, 15-min timeout.

### Option B: ECS Fargate or App Runner (Container)

1. Dockerize Claw Cubed.
2. Push image to ECR.
3. Deploy to ECS Fargate or App Runner.
4. Expose via ALB or App Runner URL.
5. **Pros:** No cold starts, WebSockets, long-lived. **Cons:** Always-on cost.

### Option C: EC2 / Lightsail (Single Instance)

1. Launch a small instance (t3.micro, Lightsail $5).
2. Run Claw Cubed binary, systemd service.
3. S3 for memory; local for sessions if desired.
4. **Pros:** Simple, predictable. **Cons:** Single point of failure.

---

## S3 Bucket Setup

```
Bucket: clawcubed-memory-{account-id}
├── memory/
│   ├── MEMORY.md
│   └── YYYYMM/
│       └── YYYYMMDD.md
```

**IAM policy** (Lambda/ECS role):

```json
{
  "Effect": "Allow",
  "Action": ["s3:GetObject", "s3:PutObject", "s3:ListBucket"],
  "Resource": [
    "arn:aws:s3:::clawcubed-memory-*",
    "arn:aws:s3:::clawcubed-memory-*/*"
  ]
}
```

---

## Environment Variables (Secrets Manager or Lambda config)

| Variable | Source |
|----------|--------|
| `PICOCLAW_MEMORY_BACKEND` | `s3` |
| `PICOCLAW_MEMORY_S3_BUCKET` | Bucket name |
| `PICOCLAW_PROVIDERS_OPENROUTER_API_KEY` | Secrets Manager |
| `TELEGRAM_BOT_TOKEN` (if used) | Secrets Manager |
| `AWS_REGION` | `us-east-1` (or your region) |

---

## Quick Start: CloudFormation / SAM / CDK

Provide a **Launch Stack** button that deploys:

1. S3 bucket for memory
2. Lambda function (or ECS task) with Claw Cubed
3. API Gateway
4. IAM roles
5. Secrets Manager placeholder for API keys

User fills in API key post-deploy; stack is ready in ~5 minutes.

---

## Cost (Rough)

| Component | Est. monthly |
|-----------|--------------|
| S3 (memory, <1GB) | ~$0.02 |
| Lambda (1M invocations) | ~$2 |
| API Gateway (1M requests) | ~$3.50 |
| Secrets Manager (1 secret) | ~$0.40 |
| **Total (light use)** | **~$6** |

Heavier use: add ECS Fargate (~$15/mo for 0.25 vCPU) or more Lambda.
