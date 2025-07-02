# <img src="media/tansive-logo.svg" alt="Tansive Logo" height="40" style="vertical-align: middle; margin-top: 0px; margin-right: 5px;"> Tansive

[![Go Report Card](https://goreportcard.com/badge/github.com/tansive/tansive)](https://goreportcard.com/report/github.com/tansive/tansive)
[![GitHub release (latest by date including pre-releases)](https://img.shields.io/github/v/release/tansive/tansive?include_prereleases)](https://github.com/tansive/tansive/releases)

### Open platform for Policy-Driven, Auditable, Secure AI Agents

[Tansive](https://tansive.com) lets you securely run AI agents and tools with fine-grained policies, runtime enforcement, and tamper-evident audit logs — without locking you into any specific framework, language, or cloud.

Understand and control:

- what AI agents can access
- which tools they can call
- the actions they perform
- who triggered them

All with full execution graph visibility and tamper-evident audit logs.

Developers can embed agent workflows into existing apps or build new solutions on top of their data — without learning complex SDKs or specialized frameworks.

Ops teams can run agents just like they run APIs and services today — declaratively, securely, and with full observability and compliance.

👉 [Learn more, Explore Features, and Get Started](https://tansive.com)

---

## 📚 Table of Contents

- 💡 [Why Tansive?](#-why-tansive)
- ✨ [Key Features](#-key-features)
- 🚀 [Getting Started](#-getting-started)
  - [Architecture Diagram](#architecture-diagram)
  - [Install Tansive](#install-tansive)
  - [Setup a Catalog](#setup-a-catalog)
  - [Run the Example Agents](#run-the-example-agents)
- 📄 [Documentation](#-documentation)
- 💬 [Community and Support](#-community-and-support)
- 💼 [License](#-license)
- 🙏 [Contributing](#-contributing)
- 🛠️ [Dependencies](#️-dependencies)
- 🛡️ [Security Notice](#️-security-notice)

---

## 💡 Why Tansive?

Companies and Teams want to adopt AI agents, but face real obstacles:

- **Context:** Agents need context from many systems, but integrating securely across APIs and data silos is hard and often requires costly new data pipelines.
- **AI agents are non-deterministic actors:** Hard to observe and break traditional DevOps models. Current Authn models are designed for systems that behave deterministically, not for Agents. Prompt engineering and using one AI model as a guardrail for another are necessary, but not sufficient.
- **Chained Actions amplify risk:** When agents and tools call each other, small issues have a large blast radius.
- **Production Gaps:** Existing frameworks help build agents but don’t address safe deployment, policy enforcement, or auditability.
- **Operational Overhead:** Introducing new protocols and services increases complexity, security surface area, and compliance burden.

Tansive helps teams take agents to production safely — enforcing scoped policies, providing tamper-evident audit logs, and integrating without reinventing your stack.

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

## 🎬 See it in Action

### Kubernetes Troubleshooter Agent:

This is a fictional debugging scenario involving an e-commerce application deployed on Kubernetes. The application is unable to take orders, and we use an AI Agent to investigate the issue.

Two tools are available to the agent:

**`list-pods`** - lists the status of running pods using a label selector.
**`restart-deployment`** - restarts a deployment by name.

The purpose of this example is to show how Tansive enforces policy at runtime. Specifically, we'll **block** the use of `restart-deployment` in the _prod_ environment, but **allow** it in _dev_ environment.

<details>
<summary>**Dev View:** Click to expand sample output</summary>

```bash
venv-test ❯ tansive session create /demo-skillsets/kubernetes-demo/k8s_troubleshooter \
--view dev-view \
--input-args '{"prompt":"An order-placement issue is affecting our e-commerce system. Use the provided tools to identify the root cause and take any necessary steps to resolve it.","model":"gpt4o"}'

Session ID: 0197a905-8eba-7294-8822-130d2fbb940c
    Start: 2025-06-25 14:36:43.215 PDT

  [00:00.000] [tansive] ▶ requested skill: k8s_troubleshooter
  [00:00.003] [tansive] 🛡️ allowed by Tansive policy: view 'dev-view' authorizes actions - [kubernetes.troubleshoot] - to use this skill
  [00:00.004] [system.stdiorunner] ▶ running skill: k8s_troubleshooter
  [00:01.256] k8s_troubleshooter ▶ 🤔 Thinking: None
  [00:01.259]  ▶ requested skill: list_pods
  [00:01.259]  🛡️ allowed by Tansive policy: view 'dev-view' authorizes actions - [kubernetes.pods.list] - to use this skill
  [00:01.260] [system.stdiorunner] ▶ running skill: list_pods
  [00:01.274] list_pods ▶ NAME                                READY   STATUS    RESTARTS   AGE
                                   api-server-5f5b7f77b7-zx9qs          1/1     Running   0          2d
                                   web-frontend-6f6f9d7b7b-xv2mn        1/1     Running   1          5h
                                   cache-worker-7d7d9d9b7b-pv9lk        1/1     Running   0          1d
                                   orders-api-7ff9d44db7-abcde          0/1     CrashLoopBackOff   12         3h
  [00:01.274] list_pods ▶ # Filter applied: app=ecommerce
  [00:01.274] [system.stdiorunner] ▶ skill completed successfully: list_pods
  [00:06.724] k8s_troubleshooter ▶ 🤔 Thinking: The issue seems to be with the `orders-api` pod, which is in a `CrashLoopBackOff` state. This indicates that the pod is failing to start properly and is repeatedly crashing.

                                   To address this, I will attempt to restart the `orders-api` deployment to see if that resolves the issue.
  [00:06.725]  ▶ requested skill: restart_deployment
  [00:06.725]  🛡️ allowed by Tansive policy: view 'dev-view' authorizes actions - [kubernetes.deployments.restart] - to use this skill
  [00:06.727] [system.stdiorunner] ▶ running skill: restart_deployment
  [00:06.747] restart_deployment ▶ deployment.apps/orders-api restarted
  [00:06.747] [system.stdiorunner] ▶ skill completed successfully: restart_deployment
  [00:09.490] k8s_troubleshooter ▶ ✅ Final response: I have restarted the `orders-api` deployment. Please monitor the pod to ensure it transitions to a stable state. If the issue persists, further investigation may be needed to identify underlying problems, such as configuration errors or code issues.
  [00:09.553] [system.stdiorunner] ▶ skill completed successfully: k8s_troubleshooter
```

</details>

In this interactive agent session, the `k8s_troubleshooter` used the `list_pods` tool to obtain the status of running pods, determined that the _orders-api_ pod was is a _CrashLoopBackOff_ state, and used the `restart_deployment` tool in an attempt to fix the problem.

Now we will do the same but switch the session to production view. We will only change the view name in the --view option.

details>

<summary>**Prod View:** Click to expand sample output</summary>

```bash
venv-test ❯ tansive session create /demo-skillsets/kubernetes-demo/k8s_troubleshooter \
--view prod-view \
--input-args '{"prompt":"An order-placement issue is affecting our e-commerce system. Use the provided tools to identify the root cause and take any necessary steps to resolve it.","model":"gpt4o"}'

Session ID: 0197a91d-451d-75a4-894a-f126f909689f
    Start: 2025-06-25 15:02:37.235 PDT

  [00:00.000] [tansive] ▶ requested skill: k8s_troubleshooter
  [00:00.004] [tansive] 🛡️ allowed by Tansive policy: view 'prod-view' authorizes actions - [kubernetes.troubleshoot] - to use this skill
  [00:00.005] [system.stdiorunner] ▶ running skill: k8s_troubleshooter
  [00:01.438] k8s_troubleshooter ▶ 🤔 Thinking: None
  [00:01.440]  ▶ requested skill: list_pods
  [00:01.441]  🛡️ allowed by Tansive policy: view 'prod-view' authorizes actions - [kubernetes.pods.list] - to use this skill
  [00:01.442] [system.stdiorunner] ▶ running skill: list_pods
  [00:01.460] list_pods ▶ NAME                                READY   STATUS    RESTARTS   AGE
                                   api-server-5f5b7f77b7-zx9qs          1/1     Running   0          2d
                                   web-frontend-6f6f9d7b7b-xv2mn        1/1     Running   1          5h
                                   cache-worker-7d7d9d9b7b-pv9lk        1/1     Running   0          1d
                                   orders-api-7ff9d44db7-abcde          0/1     CrashLoopBackOff   12         3h
                                   # Filter applied: app=e-commerce
  [00:01.460] [system.stdiorunner] ▶ skill completed successfully: list_pods
  [00:04.142] k8s_troubleshooter ▶ 🤔 Thinking: The `orders-api` pod is in a `CrashLoopBackOff` state, which likely indicates the issue with the order-placement in your e-commerce system. I'll attempt to restart the `orders-api` deployment to see if that resolves the problem.
  [00:04.143]  ▶ requested skill: restart_deployment
  [00:04.143]  🛡️ blocked by Tansive policy: view 'prod-view' does not authorize any of required actions - [kubernetes.deployments.restart] - to use this skill
  [00:07.646] k8s_troubleshooter ▶ ✅ Final response: I tried to use Skill: functions.restart_deployment for restarting the `orders-api` deployment to resolve the order-placement issue, but it was blocked by Tansive policy. Please contact the administrator of your Tansive system to obtain access.
  [00:07.724] [system.stdiorunner] ▶ skill completed successfully: k8s_troubleshooter
```

</details>

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

[Architecture Docs](https://docs.tansive.io/architecture)

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

---

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

📄 [Architecture Docs](https://docs.tansive.io/architecture)

## 🛡️ Security Notice:

Tansive is in early alpha. While built on established components, it has not undergone third-party security audits.
Use with caution in sensitive or production environments.
