![Tansive Logo](media/tansive-logo-2.png)

# Tansive

[![Go Report Card](https://goreportcard.com/badge/github.com/tansive/tansive)](https://goreportcard.com/report/github.com/tansive/tansive)
[![GitHub release (latest by date including pre-releases)](https://img.shields.io/github/v/release/tansive/tansive?include_prereleases)](https://github.com/tansive/tansive/releases)

### Open platform for Policy-Driven, Auditable, Secure AI Agents

[Tansive](https://tansive.com) lets you securely run AI agents and tools with fine-grained policies, runtime enforcement, and tamper-evident audit logs — so you can trust what they’re doing, without locking you into any specific framework, language, or cloud.

Understand and control:

- what AI agents can access
- which tools they can call
- the actions they perform
- who triggered them

All with full execution graph visibility and tamper-evident audit logs.

Developers can embed agent workflows into existing apps or build new solutions on top of their data — in their current programming language, without learning complex SDKs or specialized frameworks.

Ops teams can run agents just like they run APIs and services today — declaratively, securely, and with full observability and compliance.

Tansive is in **early alpha (0.1.0)** — not production-ready, but functional enough to explore in real workflows.

We’re gathering feedback from teams who care about integrating AI agents safely and governing them with clear policies and auditability.
If these challenges matter to you, your ideas can help shape where Tansive goes next. We invite you to check out the demos, documentation, and try Tansive out.

---

## 📚 Table of Contents

- 💡 [Why Tansive?](#-why-tansive)
- ✨ [Key Features](#-key-features)
- 🎬 [See it in Action](#-see-it-in-action)
- 📋 [What Works, What's Coming, and What to Expect](#-what-works-whats-coming-and-what-to-expect)
- 🚀 [Getting Started](#-getting-started)
  - [Architecture Diagram](#architecture-diagram)
  - [Install Tansive](#install-tansive)
  - [Setup a Catalog](#setup-a-catalog)
  - [Run the Example Agents](#run-the-example-agents)
- 📄 [Documentation](#-documentation)
- 💬 [Community and Support](#-community-and-support)
- 💼 [License](#-license)
- 🙏 [Contributing](#-contributing)
- ❓ [FAQs and Project Background](#-faqs-and-project-background)
- 🛠️ [Dependencies](#️-dependencies)
- 🛡️ [Security Notice](#️-security-notice)

---

## 💡 Why Tansive?

Companies and Teams want to adopt AI agents, but face real obstacles:

- **Context:**  
  Agents need context from many systems, but integrating securely across APIs and data silos is hard and often requires costly new data pipelines.

- **AI agents are non-deterministic actors:**  
  Hard to observe and break traditional DevOps models. Current Authn models are designed for systems that behave deterministically, not for Agents. Prompt engineering and using one AI model as a guardrail for another are necessary, but not sufficient.

- **Chained Actions amplify risk:**  
  When agents and tools call each other, small problems have a large blast radius.

- **Production Gaps:**  
  Existing frameworks help build agents but don’t adequately address safe deployment, policy enforcement, or auditability.

- **Operational Overhead:**  
  Introducing new APIs and services that speak Agent protocols increases complexity, security surface area, and compliance burden.

Tansive helps teams take agents to production safely — enforcing scoped policies, providing tamper-evident audit logs, and integrating without reinventing your stack.

---

## ✨ Key Features

- **Declarative Agent Catalog**  
  A hierarchically structured repository of agents, tools, and contextual data, partitioned across environments like dev, stage, and prod, and segmented by namespaces for teams or components.

- **Runtime Policy Enforcement**  
  Enforce fine-grained controls over access, execution, and data flows. Every invocation is checked against policy in real time.

- **Immutable Constraints and Transforms**  
  Pin runtime sessions to specific values and apply user-defined transforms to modify or redact inputs to agents and tools. Protect sensitive data (e.g. PII, Health data), apply runtime feature flags, and adapt or enrich inputs to match the expectations of your current systems without undertaking costly data migration initiatives

- **Tamper-Evident Audit Logging**  
  Maintain, hash-linked, signed logs of every action for observability, compliance, and forensic analysis.

- **Language and Framework Agnostic**  
  Author tools and agents in any language — Python, Bash, Go, Node.js — with no mandatory SDKs.

- **GitOps Friendly**  
  Configure everything via declarative YAML specs version-controlled in Git, modeled on familiar cloud-native patterns.

---

## 🎬 See it in Action

Below are examples showing how Tansive enforces policies and protects sensitive data:

**What you'll see**

- ✅ Allowing an agent to restart a deployment in dev
- 🚫 Blocking the same action in prod
- 🔒 Restricting a health bot to one patient’s records

**📺 Demo Video**: [Watch the guided walkthrough](https://vimeo.com/1099257866) (🕒 8:57)

### Example 1: Kubernetes Troubleshooter (Control agent actions via scoped Policy)

**Demonstrates**: Policy enforcement at runtime based on environment  
**Scenario**: AI agent debugging an e-commerce system  
**Key Point**: Same action allowed in _dev_, blocked in _prod_

<details> <summary>Click to expand Kubernetes Troubleshooter Example</summary>

**_Dev Environment_**

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

In this interactive agent session, the `k8s_troubleshooter` used the `list_pods` tool to obtain the status of running pods, determined that the _orders-api_ pod was is a _CrashLoopBackOff_ state, and used the `restart_deployment` tool in an attempt to fix the problem.

Now we will do the same but switch the session to production view. We will only change the view name in the --view option.

**_Prod Environment_**

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

When we switched the view to production, Tansive blocked the invocation of the `restart_deployment` tool based on the policy bound to the `prod-view`.

</details>

### Example 2: Health-Bot (Protect sensitive health data via session pinning)

**Demonstrates**: Data access control through session pinning  
**Scenario**: Health bot answering patient questions  
**Key Point**: Session locked to specific patient, blocks access to other records

<details> <summary>Click to expand Health-Bot Example</summary>

**_Access John's records but not Sheila's:_**

```bash
tansive session create /demo-skillsets/health-record-demo/health-record-agent \
--view dev-view \
--input-args '{"prompt":"John Doe and Sheila Smith were looking sick. Can you please check their bloodwork and tell me if theres anything wrong?","model":"claude"}'  \
--session-vars '{"patient_id":"H12345"}'

Session ID: 0197e74f-3f06-794e-8a1b-665f861d3586
    Start: 2025-07-07 16:53:39.878 PDT

  [00:00.000] [tansive] ▶ requested skill: health-record-agent
  [00:00.004] [tansive] 🛡️ allowed by Tansive policy: view 'dev-view' authorizes actions - [patient.labresults.get] - to use this skill
  [00:00.006] [system.stdiorunner] ▶ running skill: health-record-agent
  [00:04.479] health-record-agent ▶ 🤔 Thinking: I can help you check the bloodwork for both John Doe and Sheila Smith. I'll need to look up their information one by one.

                                   Let me start with John Doe:
  [00:04.482]  ▶ requested skill: resolve-patient-id
  [00:04.482]  🛡️ allowed by Tansive policy: view 'dev-view' authorizes actions - [patient.id.resolve] - to use this skill
  [00:04.485] [system.stdiorunner] ▶ running skill: resolve-patient-id
  [00:04.559] resolve-patient-id ▶ {
                                     "patient_id": "H12345"
                                   }
  [00:04.562] [system.stdiorunner] ▶ skill completed successfully: resolve-patient-id
  [00:08.541] health-record-agent ▶ 🤔 Thinking: Now I'll retrieve John Doe's bloodwork:
  [00:08.542]  ▶ requested skill: patient-bloodwork
  [00:08.542]  🛡️ allowed by Tansive policy: view 'dev-view' authorizes actions - [patient.labresults.get] - to use this skill
  [00:08.551] [system.stdiorunner] ▶ running skill: patient-bloodwork
  [00:08.574] patient-bloodwork ▶ {"patient_id": "H12345", "bloodwork": {"hemoglobin": 13.5, "white_cell_count": 6.2, "platelets": 250, "glucose": 98, "cholesterol": {"total": 180, "ldl": 100, "hdl": 55}}}
  [00:08.577] [system.stdiorunner] ▶ skill completed successfully: patient-bloodwork
  [00:13.690] health-record-agent ▶ 🤔 Thinking: Now let me check Sheila Smith's information:
  [00:13.691]  ▶ requested skill: resolve-patient-id
  [00:13.691]  🛡️ allowed by Tansive policy: view 'dev-view' authorizes actions - [patient.id.resolve] - to use this skill
  [00:13.692] [system.stdiorunner] ▶ running skill: resolve-patient-id
  [00:13.746] resolve-patient-id ▶ {
                                     "patient_id": "H23456"
                                   }
  [00:13.749] [system.stdiorunner] ▶ skill completed successfully: resolve-patient-id
  [00:18.895] health-record-agent ▶ 🤔 Thinking: Now I'll retrieve Sheila Smith's bloodwork:
  [00:18.896]  ▶ requested skill: patient-bloodwork
  [00:18.896]  🛡️ allowed by Tansive policy: view 'dev-view' authorizes actions - [patient.labresults.get] - to use this skill
  [00:18.897]  ❗ unable to transform input
                                    Error: Unauthorized to access patient bloodwork for patient H23456
  [00:28.640] health-record-agent ▶ ✅ Final response: Here's what I found:

                                   **John Doe (Patient ID: H12345)**
                                   Based on the bloodwork results, John Doe's values appear to be within normal ranges:
                                   - Cholesterol: Total 180 mg/dL (desirable is <200)
                                   - HDL: 55 mg/dL (optimal is >40)
                                   - LDL: 100 mg/dL (optimal is <100)
                                   - Glucose: 98 mg/dL (normal fasting is 70-99)
                                   - Hemoglobin: 13.5 g/dL (normal for adult males is typically 13.5-17.5)
                                   - Platelets: 250 × 10^3/μL (normal range is 150-450)
                                   - White cell count: 6.2 × 10^3/μL (normal range is 4.5-11.0)

                                   John's bloodwork appears normal with no concerning values.

                                   **Sheila Smith (Patient ID: H23456)**
                                   I was unable to access Sheila's bloodwork records. It appears you may not have authorization to view this patient's bloodwork information. You may need to request proper authorization to access her medical records.
  [00:28.727] [system.stdiorunner] ▶ skill completed successfully: health-record-agent

```

We locked the session to John's `patient_id` via Session Variables during Session creation. And we asked the agent to retrieve both John and Sheila's bloodwork. John's was approved but the request for Sheila's bloodwork was denied.

</details>

---

## 📋 What Works, What’s Coming, and What to Expect

Tansive is in early alpha, which means core functionality is working but there's still plenty to build. Here's what you can expect:

**What you can do today:**

You can do the following in non-production environments:

✅ Deploy agents for real workflows such as analyzing support tickets, restart failed services in dev environment, or validate data before orders are processed  
✅ Enforce policies like "This agent can only access customer data for tier 1 support cases"  
✅ Use session pinning to enforce data access controls like "This session can only access prospect data for the current lead"  
✅ Write tools in Python, Node.js, Bash or any compiled language (binary invocation)  
✅ Deploy your agent catalog and apply policies declaratively  
✅ Single User only

**What's coming:**

- Multi-User mode with project support
- Support for external Resources such as secret stores, vector DBs, conversational memory, cache
- Prometheus endpoint for observability
- Performance optimizations
- Additional security features
- Better documentation and examples

**Alpha expectations:**

- ✅ Core functionality works and is tested
- ⚠️ API may change between releases
- ⚠️ Some rough edges in the UX
- ❌ Not yet production-ready for critical systems

**Perfect for:**

- Teams experimenting with AI agents
- Proof-of-concept deployments
- Early feedback and feature requests
- Non-critical workflows

---

## 🚀 Getting Started

Read the full Installation and Getting Started guide at [docs.tansive.io](https://docs.tansive.io/getting-started)

> **Note:** Tansive is currently in **0.1-alpha** and rapidly evolving. Expect rough edges — your feedback is welcome!

### Architecture Diagram

This diagram is to set the context for the docker all-in-one quickstart image we'll use in this section. To learn about the architecture of Tansive, visit: [Concepts and Architecture](https://docs.tansive.io/concepts)

Below is a high-level view of how Tansive components connect.

```
+-----------------+      +-------------------+
|   Tansive CLI   | ---> |  Tansive Server   |
+-----------------+      +-------------------+
                                 |
                           +-------------+
                           |   Tangent   |
                           +-------------+

```

The Tansive Server acts as the control plane, coordinating policies, sessions, and audit logs. Tangent is the execution runtime that runs tools and agents. One or more Tangents can be registered with the server, and workloads are dispatched based on availability and capabilities. The CLI connects to the server for management and orchestration.

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

---

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

---

### Run the Example Agents

Run the agents shown in the "[See it in Action](#-see-it-in-action)" section.

**Run the Kubernetes Troubleshooter Agent (Control agent actions via scoped Policy)**

You don't need a cluster. The tools sends mock data.

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
--view dev-view --input-args '{"prompt":"John Doe and Sheila Smith were looking sick. Can you please check their bloodwork and tell me if theres anything wrong?","model":"claude"}' \
--session-vars '{"patient_id":"H12345"}'
```

---

### 🙏 Ready to explore?

[Get started with the docs](https://docs.tansive.io) or [start a discussion](https://github.com/tansive/tansive/discussions).

---

## 📄 Documentation

Documentation and examples are available at [https://docs.tansive.io](https://docs.tansive.io)

---

## 💬 Community and Support

Questions, Feedback, Ideas?

👉 [Start a discussion](https://github.com/tansive/tansive/discussions)

Follow us: [X](https://x.com/gettansive) | [LinkedIn](https://linkedin.com/company/tansive)

🌐 Learn more at [tansive.com](https://tansive.com)

---

## 💼 License

Tansive is Open Source under the [Apache 2.0 License](LICENSE)

## 🙏 Contributing

Contributions, issues, and feature requests are welcome.
Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

Built with care by a solo founder passionate about infrastructure, AI, and developer experience.

---

## ❓ FAQs and Project Background

#### Why is this Open Source?

Agents can do many different types of useful things for different people, teams, companies. While I've made opinionated choices on
architecture, I purposely built Tansive to be easily extensible. Trust and extensibility don't work behind closed doors. If Tansive proves it is useful and builds a community of users, there are enough layers and adjacencies that Tansive and other ecosystem participants can monetize without impacting the functionality, utility, and viability of the open ecosystem.

#### Why should we trust this project now?

Tansive is in early Alpha, and it's not ready for production use. But the foundations - hierarchical organization of agent and tool assets, policy-based views, dynamic runtime control via transforms, language agnostic runtime framework, tamper-evident logs, and extensible Resources and SkillSet abstractions - are designed to enable and sustain wide adoption of agents to automate day to day tasks without compromising on safety and compliance.

I hope you will try Tansive in your non-production environments with real workloads and provide feedback on the problems you face and the capabilities you’d like Tansive to deliver. Your insights will help shape a platform that aspires to become the standard for secure, auditable, agent-driven workflows. Thank you in advance for being part of this journey.

#### I see a large initial commit. Where is this coming from?

Tansive was developed privately by a single author [@anand-tan](https://github.com/anand-tan), and then moved to this repository to provide a clean starting point for open-source development. The [repositories](https://github.com/anand-tnsv) are publicly archived for historical reference.

## 🛠️ Dependencies:

Tansive builds on widely adopted, well-tested open-source components, including:

- Go standard library
- PostgreSQL (for catalog storage)
- Common libraries for YAML parsing, HTTP handling, and CLI UX
- No custom cryptography

Additional dependencies are listed in [`go.mod`](./go.mod)

📄 [Concepts and Architecture](https://docs.tansive.io/concepts)

## 🛡️ Security Notice:

Tansive is in early alpha. While built on established components, it has not undergone third-party security audits.
Use with caution in sensitive or production environments.

Read more: [Security notes for 0.1.0-alpha release](https://docs.tansive.io/architecture#current-alpha-limitations)
