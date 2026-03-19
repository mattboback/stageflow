# Support

Thank you for using StageFlow! If you need help, have a question, or want to report an issue, here are the ways you can get support.

## Where to get help

1. **Read the Documentation:**
   - [Architecture & Design](docs/architecture/system.md)
   - [CLI Tooling](docs/operations/devtools.md)
   - [Configuration Guide](docs/reference/configuration.md)

2. **Search Existing Issues:**
   Before creating a new issue, please search the [GitHub Issues](https://github.com/mattboback/stageflow/issues) to see if someone else has already reported your problem or asked your question.

3. **Open a New Issue:**
   If you can't find an answer, feel free to open an issue:
   - **Bug Reports:** Use the bug report template if something is broken.
   - **Feature Requests:** Use the feature request template if you have an idea for a new scanner or platform capability.
   - **Questions:** You can open an issue for general questions or usage help.

## Debugging Checklist

If you are experiencing issues running StageFlow locally or in production, please check the following before opening an issue:

- **Check Container Status:**
  ```bash
  just dev ps  # or podman ps
  ```
  Ensure all core containers (`nats`, `minio`, `postgres`, `platform-api`, `orchestrator`, `frontend`) are running and healthy.

- **Check Service Logs:**
  ```bash
  just dev logs
  ```
  Look for error messages, particularly in the `orchestrator` or `platform-api` logs.

- **Verify Environment Variables:**
  Ensure your `.env` file is properly configured based on `.env.example`. Specifically check database credentials and NATS/MinIO endpoints.

- **Network Conflicts:**
  StageFlow uses Podman networks. If containers fail to communicate, try pruning unused networks or restarting the stack:
  ```bash
  just dev down
  just dev up
  ```

## Security Issues

If you believe you have found a security vulnerability, please do **not** open a public issue. Instead, follow the instructions in our [Security Policy](SECURITY.md).
