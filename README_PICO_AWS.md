# pico-aws

A lightweight, memory-intensive AI agent built on [PicoClaw](https://github.com/sipeed/picoclaw), with **AWS S3 as the backend for agent memory**.

## What is pico-aws?

pico-aws is a variant of PicoClaw where agent memory lives in S3 instead of the local filesystem. You get:

- **Durable memory** – survives device loss, survives redeploys
- **Shared memory** – multiple instances can share the same memory (serverless, multi-node)
- **Scalable storage** – S3 handles large memory and long histories
- **Cloud-native** – fits serverless, containers, and edge deployments

## Quick Start

### 1. Build

```bash
go build -o pico-aws ./cmd/picoclaw
```

### 2. Configure S3 Memory

Set these in your config or environment:

```bash
# Memory backend: "filesystem" (default) or "s3"
PICOCLAW_MEMORY_BACKEND=s3
PICOCLAW_MEMORY_S3_BUCKET=my-pico-aws-memory
PICOCLAW_MEMORY_S3_PREFIX=          # optional, e.g. "prod/" for multi-env
PICOCLAW_MEMORY_S3_REGION=us-east-1 # optional, uses AWS_REGION if empty
```

### 3. AWS Credentials

Use standard AWS credential chain:

- `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_REGION`
- Or `aws configure` / `aws login`

### 4. S3 Bucket Setup

Create a bucket and ensure IAM has:

- `s3:GetObject`, `s3:PutObject`, `s3:ListObjectsV2` on `arn:aws:s3:::your-bucket/memory/*`

## Memory Layout in S3

- `memory/MEMORY.md` – long-term facts, preferences
- `memory/YYYYMM/YYYYMMDD.md` – daily notes

With a prefix (e.g. `prod/`): `prod/memory/MEMORY.md`, `prod/memory/202502/20250219.md`

## Running

Same as PicoClaw:

```bash
./pico-aws agent    # CLI chat
./pico-aws gateway  # HTTP gateway
# etc.
```

## Config Example

In `~/.picoclaw/config.json` or via env:

```json
{
  "memory": {
    "backend": "s3",
    "s3": {
      "bucket": "my-pico-aws-memory",
      "prefix": "",
      "region": "us-east-1"
    }
  }
}
```

## License

MIT (same as PicoClaw)
