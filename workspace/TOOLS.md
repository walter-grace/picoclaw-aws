# Tool Descriptions

This file helps the agent understand available tools. The agent has access to built-in tools and optionally AWS MCP tools when enabled.

## Built-in Tools

- **read_file** - Read files from the workspace
- **write_file** - Create or overwrite files
- **list_dir** - List directory contents
- **edit_file** - Edit files with search/replace
- **append_file** - Append content to files
- **exec** - Execute shell commands (sandboxed)
- **web_search** - Search the web for current information
- **web_fetch** - Fetch and summarize web page content
- **message** - Send a message to the user
- **spawn** - Create a subagent for async tasks
- **cron** - Schedule reminders and recurring jobs
- **shutdown_instance** - Stop the EC2 instance this agent runs on (when `tools.ec2_shutdown.enabled` is true). Requires user confirmation and IAM `ec2:StopInstances`.

## AWS MCP Tools (when enabled)

When `tools.aws_mcp.enabled` is true in config, the agent gains access to AWS MCP Server tools. These are prefixed with `aws__` and include:

- **aws__search_documentation** - Search AWS documentation and API references
- **aws__retrieve_agent_sop** - Retrieve pre-built Agent SOPs for AWS workflows
- **aws__call_api** - Execute AWS API calls with SigV4 authentication

Additional tools may be available depending on the AWS MCP Server configuration. Use these tools for:

- Provisioning and configuring AWS infrastructure
- Troubleshooting AWS issues (CloudWatch, CloudTrail)
- Managing costs and billing
- Executing multi-step AWS workflows following best practices

**Prerequisites:** AWS credentials (`aws configure` or `aws login`), uv/uvx for mcp-proxy-for-aws.
