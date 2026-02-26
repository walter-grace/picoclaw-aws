# Claw Cubed Chat UI

A web chat interface for Claw Cubed, inspired by the [Vercel AI Chatbot](https://github.com/vercel/ai-chatbot) pattern.

## Architecture

```
┌─────────────────┐     POST /api/chat      ┌─────────────────┐
│   Next.js Web   │ ──────────────────────► │  Claw Cubed     │
│   (port 3000)   │     { messages }        │  Gateway        │
│                 │ ◄────────────────────── │  (port 18790)   │
│                 │     { content }         │                 │
└─────────────────┘                         └─────────────────┘
```

## Quick Start

**Terminal 1 – Claw Cubed gateway:**

```bash
cd /path/to/claw-cubed
./clawcubed gateway
```

**Terminal 2 – Web UI:**

```bash
cd /path/to/claw-cubed/web
npm install
npm run dev
```

Open [http://localhost:3001](http://localhost:3001).

## HTTP Chat API

Claw Cubed gateway exposes:

- `POST /api/chat` – JSON body `{ messages: [{ role, content }] }`, returns `{ content }`
- `POST /api/chat/stream` – SSE stream (for future streaming UI)

## Upgrading to Full Vercel AI Chatbot

To use the full [Vercel ai-chatbot](https://github.com/vercel/ai-chatbot) template:

1. Clone: `npx degit vercel/ai-chatbot web-full`
2. Add an API route that proxies to `http://localhost:18790/api/chat`
3. Or add a custom transport that calls the Claw Cubed backend
