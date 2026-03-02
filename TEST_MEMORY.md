# pico-aws Memory Test Guide

## S3 Memory (Default)

pico-aws uses **S3 as the default memory backend**. No local `MEMORY.md` file—the bucket is the memory.

1. **Create an S3 bucket** (one-time):
   ```bash
   aws s3 mb s3://your-pico-aws-memory --region us-east-1
   ```

2. **Set the bucket** in `~/.picoclaw/config.json` or env:
   ```bash
   export PICOCLAW_MEMORY_S3_BUCKET=your-pico-aws-memory
   ```
   Or in config:
   ```json
   "memory": {
     "backend": "s3",
     "s3": {
       "bucket": "your-pico-aws-memory",
       "prefix": "",
       "region": "us-east-1"
     }
   }
   ```

3. **Run the agent** – memory goes to S3:
   ```bash
   ./run.sh agent -m "Remember this in S3: test fact 123"
   ```

4. **Verify in S3**:
   ```bash
   aws s3 ls s3://your-pico-aws-memory/memory/
   aws s3 cp s3://your-pico-aws-memory/memory/MEMORY.md -
   ```

## Fallback to Filesystem

If S3 bucket is not set, pico-aws falls back to local files (`~/.picoclaw/workspace/memory/MEMORY.md`).

## Test S3 Connectivity

Run the test script to verify the agent can reach S3:

```bash
cd picoclaw-aws && go run ./scripts/test_s3_memory.go
# or
./scripts/test_s3_memory.sh
```

The script checks: config, AWS credentials, bucket access, and read/write round-trip.

## Verified Flow

- Agent reads `MEMORY.md` (via system prompt + read_file)
- Agent writes/edits memory via `edit_file` or `write_file`
- Memory persists across sessions
- Recall works: ask "What do you remember about me?"
