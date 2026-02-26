# Prompt: Build a Website for Deploying Claw Cubed on AWS

Use this prompt with an AI coding assistant (e.g., Cursor, Claude, GPT) to build the deployment website.

---

## The Prompt

```
Build a website that lets users deploy their own Claw Cubed instance on AWS with one click.

### Requirements

1. **Landing page**
   - Hero section: "Deploy your own AI agent in minutes"
   - Short description of Claw Cubed (lightweight agent, S3 memory, cloud-native)
   - Clear CTA: "Deploy to AWS" button

2. **Deploy flow**
   - User clicks "Deploy to AWS"
   - Redirect to AWS CloudFormation "Launch Stack" with a pre-built template
   - OR: Show a form to collect:
     - AWS region (dropdown)
     - LLM API key (OpenRouter/OpenAI) – stored in Secrets Manager
     - Optional: Telegram/Discord token
     - Optional: S3 bucket name (or auto-generate)
   - After form submit: either
     - (A) Open CloudFormation console with parameters pre-filled via URL
     - (B) Call AWS (via backend) to create stack – requires backend with AWS creds
     - (C) Generate a CloudFormation/SAM/CDK template and provide "Download" + "Launch Stack" link

3. **Post-deploy**
   - Show "Your agent is deploying" with link to CloudFormation stack
   - Link to docs: how to get the API URL, how to connect Telegram, etc.
   - Optional: status check (poll stack status, show "Ready" when done)

4. **Tech stack**
   - Static site (Next.js, Astro, or plain HTML/JS) for the landing page
   - No backend required if using CloudFormation Launch Stack URL with parameters
   - If backend: use Next.js API routes or a small Lambda to invoke CloudFormation

5. **Design**
   - Clean, minimal, developer-friendly
   - Dark or light theme
   - Mobile responsive

### AWS artifacts to create (separate task)

- CloudFormation or SAM template that:
  - Creates S3 bucket for memory
  - Creates Lambda (or ECS) with Claw Cubed container
  - Creates API Gateway
  - Creates IAM role
  - Creates Secrets Manager secret (empty, user fills via console)
  - Outputs: API URL, WebSocket URL (if applicable)

### Deliverables

1. Landing page + deploy flow (frontend)
2. CloudFormation/SAM template for Claw Cubed
3. README with: how to run locally, how to deploy the website itself, how the "Deploy to AWS" flow works
```

---

## Simplified Variant (No Backend)

If you want a **static site only** (no server, no AWS SDK in the browser):

```
Build a static landing page for Claw Cubed with:

1. Hero: "Deploy your own Claw Cubed on AWS"
2. "Deploy to AWS" button that opens:
   https://console.aws.amazon.com/cloudformation/home#/stacks/new?stackName=clawcubed&templateURL=https://...template.json

3. A short form (client-side only) that builds the CloudFormation parameter URL:
   - User enters: API key, region, bucket name
   - JS builds the Launch Stack URL with query params
   - Opens in new tab

4. Docs section: "After deploy, your agent URL is..." with link to CloudFormation outputs

5. Use Tailwind or similar. Minimal, clean design.
```

---

## What You Need to Provide

- **CloudFormation template** – Create this first (or use the AWS one-pager as a spec)
- **S3/CloudFront URL** for the template if hosting it yourself
- **Repo structure** – e.g. `website/` folder in claw-cubed for the landing page
