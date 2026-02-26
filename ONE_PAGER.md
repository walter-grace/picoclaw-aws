# pico-aws: The AWS-Native Tiny Agent

## What It Is

**pico-aws** is an improved, AWS-powered variant of PicoClaw. It keeps the same lightweight design but is built for the cloud: S3 for memory, AWS MCP for tools, and a deployment model that scales.

---

## pico-aws vs PicoClaw

| | PicoClaw | pico-aws |
|---|----------|------------|
| **Memory** | Local `MEMORY.md` file | **S3 bucket** – no local file, cloud-backed |
| **AWS tools** | Optional (off by default) | **AWS MCP enabled by default** – call APIs, search docs, run workflows |
| **Durability** | Tied to one machine | Survives restarts, redeploys, device loss |
| **Sharing** | Single instance | Multiple instances share the same S3 memory |
| **Deployment** | Single node, local disk | Serverless (Lambda), containers (ECS), multi-node |
| **Identity** | General-purpose tiny agent | **AWS-native, scalable, cloud-first** |

---

## Unique Features

1. **S3 as memory** – Use one **dedicated** S3 bucket for agent memory only (MEMORY.md, daily notes). Create and use **other buckets** for media and assets. No `MEMORY.md` on disk; durable, shared, and scalable.
2. **AWS MCP by default** – The agent can use AWS APIs, search documentation, and run AWS workflows out of the box.
3. **Cloud-native** – Designed for Lambda, ECS, and stateless deployments. Memory lives in S3; instances can come and go.
4. **Same core** – Same channels (Telegram, Discord, etc.), same skills, same lightweight footprint. Just better infrastructure.

---

## Our Take on Tiny Agents

**Small agents, big reach.** PicoClaw proved a tiny agent can be capable. pico-aws pushes that further:

- **Lightweight** – Low RAM, low CPU. Runs on cheap hardware and serverless.
- **AWS-powered** – S3 memory + MCP tools. Use AWS without extra glue.
- **Durable** – Memory in S3 survives crashes and redeploys.
- **Shared** – One memory store for many instances. Team bots, handoffs, HA.
- **Scalable** – Stateless compute + S3 memory = horizontal scaling.

---

## In One Line

**pico-aws = PicoClaw + S3 memory + AWS MCP by default. Same agent, AWS-native and scalable.**
