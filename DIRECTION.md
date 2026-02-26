# Claw Cubed: Direction & Vision

## Core Idea

Claw Cubed is **PicoClaw, improved for AWS**. Same tiny agent, but:

- **S3 is the memory** – No local file. The bucket is the brain.
- **AWS MCP by default** – The agent can call AWS APIs, search docs, run workflows from day one.
- **Cloud-native** – Built for Lambda, ECS, serverless. Stateless compute + S3 memory = scalable.

## What We Have Today

| Feature | Status |
|---------|--------|
| S3 memory backend | Done – default |
| AWS MCP enabled by default | Done |
| Fallback to filesystem when no bucket | Done |
| Same channels, skills, tools as PicoClaw | Done |

## Future: Making It More High-Functioning

Ideas to push Claw Cubed further:

1. **Bedrock as LLM provider** – Use Claude on Bedrock. Fully AWS-native, no external API keys.
2. **RAG over S3** – Index documents in S3, semantic search for context. Memory that scales beyond one file.
3. **DynamoDB for structured memory** – Facts, entities, preferences in a queryable store.
4. **Multi-tenant** – S3 prefix per user (`memory/user-123/`) for isolated memory.
5. **Deploy templates** – CloudFormation/SAM for one-click deploy to Lambda or ECS.
6. **CloudWatch integration** – Structured logs, metrics for observability.

## Positioning

- **PicoClaw** = Tiny agent, local-first, general-purpose.
- **Claw Cubed** = Tiny agent, AWS-first, scalable, cloud-native.

Same DNA. Different habitat.
