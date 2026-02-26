# Security

## Reporting a vulnerability

If you discover a security vulnerability, please report it responsibly:

1. **Do not** open a public issue.
2. Email the maintainers or open a private security advisory on GitHub.
3. Include steps to reproduce and impact assessment if possible.

We will acknowledge and respond as soon as we can.

## Security checklist for deployments

- [ ] `.env` is not in the repo and not committed
- [ ] Secrets are stored in AWS Secrets Manager or Parameter Store (recommended)
- [ ] Instance has minimal outbound access (Telegram, OpenRouter, AWS APIs)
- [ ] No unnecessary ports open
- [ ] Logs do not contain API keys or tokens
