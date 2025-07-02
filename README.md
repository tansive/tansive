# <img src="media/tansive-logo.svg" alt="Tansive Logo" height="40" style="vertical-align: middle; margin-top: 0px; margin-right: 5px;"> Tansive

[![Go Report Card](https://goreportcard.com/badge/github.com/tansive/tansive)](https://goreportcard.com/report/github.com/tansive/tansive)

### Open platform for Policy-Driven, Auditable, Secure AI Agents

Tansive lets you securely run AI agents and tools with fine-grained policies, runtime enforcement, and tamper-evident audit logs — without locking you into any specific framework, language, or cloud.

Understand and control:

- what AI agents can access
- which tools they can call
- the actions they perform
- who triggered them

All with full execution graph visibility and tamper-evident audit logs.

Developers can embed agent workflows into existing apps or build new solutions on top of their data — without learning complex SDKs or specialized frameworks.

Ops teams can run agents just like they run APIs and services today — declaratively, securely, and with full observability and compliance.

[👉 Learn more, Explore Features and Get Started](https://tansive.com)

---

## 📚 Table of Contents

- [✨ Key Features](#-key-features)
- [🚀 Getting Started](#-getting-started)
  - [Architecture Diagram](#architecture-diagram)
  - [Install Tansive](#install-tansive)
  - [Setup a Catalog](#setup-a-catalog)
  - [Run the Example Agents](#run-the-example-agents)
- [📄 Documentation](#-documentation)
- [💬 Community and Support](#-community-and-support)
- [💼 License](#-license)
- [🙏 Contributing](#-contributing)
- [🛠️ Dependencies](#️-dependencies)
- [🛡️ Security Notice](#️-security-notice)

---

## ✨ Key Features

- **Declarative Agent Catalog**  
  A hierarchically structured repository of agents, tools, and contextual data, partitioned across environments like dev, stage, and prod, and segmented by namespaces for teams or components.

- **Runtime Policy Enforcement**  
  Enforce fine-grained controls over access, execution, and data flows. Every invocation is checked against policy in real time.

- **Immutable Constraints and Transforms**  
  Pin runtime sessions to specific values and apply user-defined transforms to modify or redact inputs to agents and tools. Protect sensitive data (e.g. PII, Health data) and apply runtime feature flags.

- **Tamper-Evident Audit Logging**  
  Maintain, hash-linked, signed logs of every action for observability, compliance, and forensic analysis.

- **Language and Framework Agnostic**  
  Author tools and agents in any language — Python, Bash, Go, Node.js — with no mandatory SDKs.

- **GitOps Friendly**  
  Configure everything via declarative YAML specs version-controlled in Git, modeled on familiar cloud-native patterns.

---

## 🚀 Getting Started

Read the full Installation and Getting Started guide at [docs.tansive.io](https://docs.tansive.io/getting-started)

> **Note:** Tansive is currently in **0.1-alpha** and rapidly evolving. Expect rough edges — your feedback is welcome!

### Architecture Diagram

Below is a high-level view of how Tansive components connect:

```
+-----------------+      +-------------------+
|   Tansive CLI   | ---> |  Tansive Server   |
+-----------------+      +-------------------+
                                 |
                           +-------------+
                           |   Tangent   |
                           +-------------+

```

The CLI connects to the Tansive Server, which orchestrates runtime sessions via Tangent.

### Install Tansive

1. **Run the Tansive Server and Tangent**

```bash
docker compose -f scripts/docker/docker-compose-aio.yaml up -d
```

Wait for the `tangent` service to reach the `started` state. Use `--pull always` option if you have already run Tansive and need to get the latest images.

2. **Install the CLI**

Download the appropriate release binary named `tansive-<version>-<os>-<arch>.tar.gz` from [Releases](https://github.com/tansive/tansive/releases) or build from source.

```bash
# Verify CLI installation
tansive version

# Configure CLI
tansive config --server https://local.tansive.dev:8678

# Login in single user mode
tansive login

# Verify status
tansive status
```

### Setup a Catalog

1. **Configure API Keys**

Configure API Keys to run the sample agents.

Create a `.env` file in the project root with your OpenAI or Anthropic API Key. Replace only the API key you want to use - keep the placeholder for keys you don't need (e.g., if you only use Claude, keep `<your-openai-key-here>` as-is).

```bash
# Create the .env file
# Replace only the API keys you want to use - keep placeholders for unused keys
cat > .env << 'EOF'
CLAUDE_API_KEY="<your-claude-key-here>"
OPENAI_API_KEY="<your-openai-key-here>"

# Note: You don’t need an actual Kubernetes cluster. This dummy config is used by the demo agent.
KUBECONFIG="YXBpVmVyc2lvbjogdjEKa2luZDogQ29uZmlnCmNsdXN0ZXJzOgogIC0gbmFtZTogbXktY2x1c3RlcgogICAgY2x1c3RlcjoKICAgICAgc2VydmVyOiBodHRwczovL2Rldi1lbnYuZXhhbXBsZS5jb20KICAgICAgY2VydGlmaWNhdG9yaXR5LWRhdGE6IDxiYXNlNjQtZW5jb2RlZC1jYS1jZXJ0Pg=="
EOF
```

2. **Setup the example Catalog via declarative scripts**

```bash
# sets up a new catalog with dev and prod variants, and SkillSets
bash examples/catalog_setup/setup.sh

# view the catalog structure that was setup
tansive tree
```

**Quick smoke test:** Run `tansive tree` to verify the Catalog was set up correctly.

### Run the Example Agents

**Run the Ops Troubleshooter Agent (Control agent actions via scoped Policy)**

Change `model` to "gpt4o" or "claude" depending on the API Key

```bash
# Run in 'dev' environment (agent will redeploy a pod)
tansive session create /demo-skillsets/kubernetes-demo/k8s_troubleshooter \
--view dev-view \
--input-args '{"prompt":"An order-placement issue is affecting our e-commerce system. Use the provided tools to identify the root cause and take any necessary steps to resolve it.","model":"claude"}'


# Run in 'prod' environment. (policy will block redeployment)
tansive session create /demo-skillsets/kubernetes-demo/k8s_troubleshooter \
--view prod-view \
--input-args '{"prompt":"An order-placement issue is affecting our e-commerce system. Use the provided tools to identify the root cause and take any necessary steps to resolve it.","model":"claude"}'
```

**Run the Health Bot Agent (Protect sensitive PHI data via session pinning)**

```bash
# Run the Health Bot with Session pinned to John's patient_id
tansive session create /demo-skillsets/health-record-demo/health-record-agent \
--view dev-view \
--input-args '{"prompt":"John was looking sick. Can you please check his bloodwork?","model":"gpt4o"}' \
--session-vars '{"patient_id":"H12345"}'


# Try to fetch Sheila's record (expected: Tansive will reject this request)
tansive session create /demo-skillsets/health-record-demo/health-record-agent \
--view dev-view \
--input-args '{"prompt":"Sheila was looking sick. Can you please check her bloodwork?","model":"gpt4o"}' \
--session-vars '{"patient_id":"H12345"}'

```

### 🙏 Ready to explore?

[Get started with the docs](https://docs.tansive.io) or [start a discussion](https://github.com/tansive/tansive/discussions).

## 📄 Documentation

Documentation and examples are available at [https://docs.tansive.io](https://docs.tansive.io)

## 💬 Community and Support

Questions, Feedback, Ideas?

👉 [Start a discussion](https://github.com/tansive/tansive/discussions)

Follow us:

[X](https://x.com/gettansive) | [LinkedIn](https://linkedin.com/company/tansive)

🌐 Learn more at [tansive.com](https://tansive.com)

## 💼 License

Tansive is Open Source under the [Apache 2.0 License](LICENSE)

## 🙏 Contributing

Contributions, issues, and feature requests are welcome.
Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

Built with care by a solo founder passionate about infrastructure, AI, and developer experience.

Note: Tansive was developed privately from March–July 2025 and released as open source in this public repository. Earlier development history was retained privately to maintain a clear and focused public history.

## 🛠️ Dependencies:

Tansive builds on widely adopted, well-tested open-source components, including:

- Go standard library
- PostgreSQL (for catalog storage)
- Common libraries for YAML parsing, HTTP handling, and CLI UX
- No custom cryptography

Additional dependencies are listed in [`go.mod`](./go.mod)

📄 [Architecture Docs](https://docs.tansive.io/architecture.md)

## 🛡️ Security Notice:

Tansive is in early alpha. While built on established components, it has not undergone third-party security audits.
Use with caution in sensitive or production environments.
