![Tansive Logo](media/tansive-logo-2.png)

# Tansive

[![Go Report Card](https://goreportcard.com/badge/github.com/tansive/tansive)](https://goreportcard.com/report/github.com/tansive/tansive)
[![GitHub release (latest by date including pre-releases)](https://img.shields.io/github/v/release/tansive/tansive?include_prereleases)](https://github.com/tansive/tansive/releases)

### Platform for Secure and Policy-driven AI Agents

[Tansive](https://tansive.com) lets you securely run AI agents and tools with fine-grained policies, runtime enforcement, and tamper-evident audit logs.

Understand and control:

- What AI agents access
- Which tools they can call
- The actions they perform
- Who triggered them

All with full execution graph visibility and audit logs.

Tansive lets developers run AI agents with controlled access to tools — you define what tools each agent can use, specify runtime validation of tool inputs, and Tansive enforces those rules while logging full tool call traces.

```bash
$ tansive session create /skillsets/tools/deployment-tools --view devops-engineer
Session created. MCP endpoint:
https://127.0.0.1:8627/session/mcp
Access token: tn_7c2e4e0162df66d929666703dc67a87a
```

For Ops Teams, Tansive provides a `yaml` based control plane for running AI agents and tools with policy-driven security and full observability.

```yaml
#This example specifies Allow/Deny rules for a Kubernetes troubleshooter agent
spec:
  rules:
    - intent: Allow
      actions:
        - system.skillset.use
        - kubernetes.pods.list
        - kubernetes.troubleshoot
      targets:
        - res://skillsets/agents/kubernetes-agent
    - intent: Deny
      actions:
        - kubernetes.deployments.restart
      targets:
        - res://skillsets/agents/kubernetes-agent
```

---

## 📚 Table of Contents

- 💡 [Why Tansive?](#-why-tansive)
- ✨ [Key Features](#-key-features)
- 🔑 [Key Concepts](#-key-concepts)
- 🔧 [How Tansive works](#-how-tansive-works)
- 🧱 [Architecture](#-architecture)
- 🎬 [See it in Action](#-see-it-in-action)
- 📋 [What Works, What's Coming, and What to Expect](#-what-works-whats-coming-and-what-to-expect)
- 🚀 [Getting Started](#-getting-started)
  - [Overview](#overview)
  - [Install Tansive](#installation)
  - [Setup a Catalog](#setup-a-catalog)
  - [Run the Example Agents](#run-the-examples)
    - [Set up a policy-governed MCP proxy with input filtering for Github (or your favorite MCP server)](#set-up-a-policy-governed-mcp-proxy-with-input-filtering-for-github-or-your-favorite-mcp-server)
    - [Run the Policy-driven agent](#run-the-policy-driven-agent)
    - [Run the Secure Data Handling example](#run-the-secure-data-handling-example)
  - [Explore the Audit Log](#explore-the-audit-log)
- 📄 [Documentation](#-documentation)
- 💬 [Community and Support](#-community-and-support)
- 💼 [License](#-license)
- 🙏 [Contributing](#-contributing)
- ❓ [FAQs and Project Background](#-faqs-and-project-background)
- 🛠️ [Dependencies](#️-dependencies)
- 🛡️ [Security Notice](#️-security-notice)

---

## 💡 Why Tansive?

- **Secure Integration**: AI Tools and Agents need context from many systems, but integrating securely across different APIs is challenging. Tansive provides rules-based access control at every interface in the Agent pipeline.

- **Reduce Operational Burden**: Deploy and manage AI Agents using the same GitOps processes teams use today. Tansive is cloud-agnostic and works across clouds.

- **Chained Actions amplify risk:** When agents and tools call each other, small problems have a large blast radius. Tansive provides audit logs that capture the full tool call graph and policy decisions to help trace and mitigate risks.

- **Defend against Security Vulnerabilities**: AI tools introduce new attack vectors like prompt injections. Tansive provides the tools to implement defense-in-depth security processes to keep systems safe.

- **Meet Compliance Requirements**: Systems often have regulatory requirements to meet (SOC2, HIPAA, PCI, Data Privacy). Tansive provides policy-based control and tamper-evident logs for compliance and audits.

---

## ✨ Key Features

- **Declarative Agent Catalog**  
  A hierarchically structured repository of agents, tools, and contextual data, partitioned across environments like dev, stage, and prod, and segmented by namespaces for teams or components.

- **Runtime Policy Enforcement**  
  Enforce fine-grained controls over tool calls. Every invocation is checked against policy in real time.

- **Runtime Constraints and Transforms**  
  Pin runtime sessions to specific values and apply user-defined transforms to modify or redact inputs to agents and tools. Protect sensitive data (e.g. PII, Health data), apply runtime feature flags, and adapt or enrich inputs to match the expectations of your current systems without undertaking costly data migration initiatives.

- **Audit Logging**  
  Maintain hash-linked, signed logs of every action for observability, compliance, and analysis.

- **Agents in any framework**  
  Author agents in any framework - LangGraph, CrewAI, Semantic Kernel, etc.

- **Tools in any language**  
  Author tools in any language or import tools from local or external MCP servers. Tansive will run or proxy them and create a policy-governed MCP endpoint with secure HTTP transport.

- **GitOps Friendly**  
  Configure everything via declarative YAML specs version-controlled in Git, modeled on familiar cloud-native patterns.

---

## 🔑 Key Concepts

Before diving into how Tansive works, here are the core concepts you'll encounter:

- **SkillSet**: A collection of tools and agents that work together to accomplish a specific type of task. Think of it as a template that defines what tools are available and how they should be run.
- **View**: A policy that controls what actions are allowed in a session. Views define which capabilities (like `kubernetes.deployments.restart` or `patient.labresults.get`) are permitted.
- **Session**: A runtime instance of a SkillSet with specific constraints applied. When you create a session, Tansive instantiates the SkillSet and applies the View policy to control access.
- **MCP (Model Context Protocol)**: A standard protocol for AI tools to communicate with language models. Tansive creates secure MCP endpoints that expose your tools as defined by policy and applies runtime input validation and constraints.
- **Skill**: Individual tools or agents within a SkillSet. Skills can be written in any language or framework and are the building blocks of your workflows.
- **Capability**: Granular permissions expressed as tags (e.g., `query.sql.select`, `payments.reconcile`) that define what actions are allowed.

---

## 🔧 How Tansive works

#### For Developers:

Tansive lets developers run AI agents with controlled access to tools — you define what tools each agent can use, and Tansive enforces those rules while logging full tool call traces.

- Call tools via Tansive from your agents written in LangGraph, CrewAI or any of your favorite frameworks, so you get filtered tool access, runtime evaluation of tool inputs, and detailed audit logs with full tool call lineage.

- Tansive is also an orchestrator, running your tools or agent code directly.

  - Let Tansive run your tools written in Python, Node, Go, etc. Tansive will automatically create an MCP endpoint for your tools with authenticated HTTP transport, so you don't have to manage tokens and authorization.
  - You can also have Tansive run your agent code directly. When you do, the agent will be subject to the same policy and runtime access constraints, giving you end-to-end control over the agent.

- Run multiple concurrent sessions with different policies — Create a template with a collection tools and agents tagged with _Capability_ tags, then create filters based on those tags to control what each session can do. In Tansive, a collection of tools and agents that can together accomplish a type of job is called a `SkillSet`. Policies are called `Views`.

<details> <summary>Examples: Create tool and agent sessions</summary>

e.g., You can create secure MCP endpoints for different roles to configure tools such as cursor.

```bash
$ tansive session create /skillsets/tools/deployment-tools --view devops-engineer
Session created. MCP endpoint:
https://127.0.0.1:8627/session/mcp
Access token: tn_7c2e4e0162df66d929666703dc67a87a

```

e.g., You can run a complete agent workflow with a prompt. You can also restrict the agent from accessing sensitive data.

```
$ tansive session create /skillsets/agents/health-record-agent \
--view prod-view \
--input-args '{"prompt":"John Doe and Sheila Smith were looking sick. Can you please check their
bloodwork and tell me if there is anything wrong?","model":"claude"}' \
--session-vars '{"patient_id":"H12345"}' --interactive

# This agent session runs the agent bot with the provided prompt. We scoped the session to John's patient_id
# so while the agent can access John's data, it cannot access Sheila's.

```

</details>

#### For DevOps Engineers:

Tansive provides a `yaml` based control plane for running AI agents and tools with policy-driven security and observability.

- Create a `Catalog` of workflows consisting of Tools and Agents by defining them in YAML
- Assign them across different environments such as `dev`, `stage`, `prod`. Give teams their own `namespace` in each environment
- Define access and runtime policies in YAML based on environments and namespaces
- Run workflows scoped to the policies you created.
- Run your tools and agents across different VPCs depending on systems they need to access.
- Deploy and manage AI Agents using your existing CI/CD processes.

<details> <summary>Example `yaml` definition of a SkillSet</summary>

```yaml
spec:
  version: "0.1.0"
  sources:
    - name: resolve-patient-id
      runner: "system.stdiorunner"
      config:
        version: "0.1.0-alpha.1"
        runtime: "node"
        script: "resolve-patient-id.js"
        security:
          type: default
    - name: patient-bloodwork
      runner: "system.stdiorunner"
      config:
        version: "0.1.0-alpha.1"
        runtime: "python"
        script: "patient_bloodwork.py"
        security:
          type: default
    - name: agent-script
      runner: "system.stdiorunner"
      config:
        version: "0.1.0-alpha.1"
        runtime: "python"
        script: "run-llm.py"
        security:
          type: default
  context:
    - name: claude
      schema:
        type: object
        properties:
          apiKey:
            type: string
          model:
            type: string
        required:
          - apiKey
          - model
      value:
        apiKey: { { .ENV.CLAUDE_API_KEY } }
        model: claude-3-7-sonnet-latest
    - name: gpt4o
      schema:
        type: object
        properties:
          apiKey:
            type: string
          model:
            type: string
        required:
          - apiKey
          - model
      value:
        apiKey: { { .ENV.OPENAI_API_KEY } }
        model: gpt-4o
  skills:
    - name: resolve-patient-id
      source: resolve-patient-id
      description: "Resolve patient ID"
      inputSchema:
        type: object
        properties:
          name:
            type: string
            description: "Patient name"
        required:
          - name
      outputSchema:
        type: string
        description: "Patient ID"
      exportedActions:
        - patient.id.resolve
      annotations:
        llm:description: |
          Resolve the patient ID from the patient's name.
          This skill is used to resolve the patient ID from the patient's name.
          It requires the patient name as input and will return the patient ID in json.
```

</details>

---

## 🧱 Architecture

### Functional Architecture of a Tansive Session

This diagram shows how Tansive orchestrates tools, policies, and agents within a session runtime

![Tansive Functional Architecture](media/tansive-functional-arch.svg)

When you create a session, Tansive creates a runtime instance of the "SkillSet" constrained by the specified "View". SkillSet is a `yaml` template that specifies all the tools and agents involved in accomplishing a specific type of task (e.g., placing an order based on availability of inventory) along with information on how to run them, and what capabilities they expose.  
A "View" is a policy that specifies what capabilities are permitted to be used in a session. Capabilities are expressed via tags such as `kubernetes.deployments.restart`, `query.sql.update`, that you define at your desired level of granularity.

At the core of a Tansive session is the Tool Call router. Tools can be added to the router from various sources:

- **Remote MCP Servers:** MCP endpoints hosted by external resources that use HTTP Transport (Remote MCP)
- **Local MCP Servers:** MCP servers that use `stdio` transport that access external resources via REST API or any other means such as gRPC. (Local MCP)
- **Local Scripts:** Simple run-once-and-exit tools can be written in any language. These tools are run by Tansive with a json argument and the tool prints results to standard output or error.

When a session is created, Tansive automatically creates an MCP endpoint using authenticated HTTP transport.

- Only the Tools specified by the View policy associated with the session are exposed via the endpoint.
- The router applies tool input transformations specified in the SkillSet at runtime before tool calls are dispatched.
- All tool calls, their inputs, and policy evaluation decisions and rules that supported the decision are logged.

Agents can integrate with Tansive in one of two modes:

- Tansive can run the agent directly subjecting it to the same View policies, therefore providing end to end control over the entire pipeline. In this case, the agent accesses tools via the Tansive SkillSet service which uses a Unix Domain Socket as transport. Upcoming releases will add support for Tansive managed agents to access tools via MCP.

- Agents are run independently, but Tansive governs tool access. This mode also covers agents in tools such as Cursor, Claude desktop, etc.

---

## 🎬 See it in Action

### Policy-governed secure MCP tools proxy for IDEs such as Cursor, Copilot, Claude, etc

If you are using tools like Cursor, Copilot, Claude, etc with MCP servers such as Github's, [read this article](https://docs.tansive.io/blog/implementing-defense-prompt-injection-attacks-mcp) on how Tansive can be used to set up use-case or role based policies to defend against unintended agent actions or prompt injection vulnerabilities.

### Examples of secure, policy-driven agents

Below examples show how Tansive enforces policies and protects sensitive data:

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

Tansive is in alpha and under active development. Here's what you can expect:

**What you can do today:**

You can do the following in non-production environments:

✅ Deploy policy-based filter and input validation for MCP tools that you use with Cursor, Claude, Windsurf, etc. You can create them under different roles that you can toggle in cursor.

- Only tools with `stdio` transport are supported in this release - which includes a large number of tools. See Roadmap below for other transports.
- You can also write your own MCP tools using `stdio` transport and Tansive will run it.
- Tansive will automatically create an MCP endpoint with secure HTTP transport and bearer token authentication that you can configure in your IDEs.

✅ Deploy interactive agents for real workflows such as analyzing support tickets, restart failed services in dev environment, or validate data before orders are processed.  
✅ Enforce policies like "This agent can only access customer data for tier 1 support cases"  
✅ Use session pinning to enforce data access controls like "This session can only access prospect data for the current lead"  
✅ Write simple run-once-and-exit tools in Python, Node.js, Bash or any compiled language. In the current release, these tools are only exposed to Agents via the Tansive SkillSet service - an internal toolcall interface.  
✅ Deploy your agent catalog, write and apply policies declaratively in `yaml`.  
✅ Single User only

**What's coming:**

Target: Mid August, 2025:

- All tools are exported via Tansive's secure MCP endpoint as well as Tansive's SkillSet interface.
- Support for multiple MCP servers in one Tansive SkillSet. This allows one to compose rich workflows combining multiple systems.
- Support for remote MCP servers
- Better documentation and examples

Target: Mid September, 2025:

- Support for multiple users
- Secure onboarding of Tangents

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

### Overview

- You'll install Tansive in just a few minutes
- Run two quick examples
  - **Policy-Driven Agent**: Watch how Tansive controls agent and tool invocations in real time with fine-grained policies. Instantly see which actions are allowed or denied, and why.
  - **Secure Data Handling**: Learn how Tansive protects sensitive information using pinned session variables working alongside a user-defined transform.
- If you just want to learn how Tansive works without installing anything, skip ahead to [Setup a Catalog](#setup-a-catalog). The output of all hands-on examples is provided so you can easily follow along.

### Installation

🕒 Total time: 5-7 minutes

#### Prerequisites

- Latest stable version of Docker desktop or Docker Engine [🔗](https://www.docker.com/get-started/)
- Git
- **Platform Support :**
  - **macOS** (Apple Silicon and x86_64) and **Linux** (x86_64 and arm64) are fully supported and tested.
  - on **Windows 11:**
    - Tansive CLI client is supported natively. Git Bash or WSL2 is recommended.
    - Tansive Server and Runtime are supported under WSL2
  - Docker containers
- There are no additional requirements for running the policy-governed MCP proxy example.
- An OpenAI or Anthropic API key is required to run the agent examples.
  - The example scripts use the OpenAI Python SDK.
  - Each run consumes approximately 10,000 input tokens and 1,000 output tokens.
- No API key? No Problem. Sample real output is provided, so you can follow along without running the examples.

Have questions or need help? [Post your question here](https://github.com/tansive/tansive/discussions/categories/need-help-getting-started-with-tansive)

#### Installation Steps

#### **1. Clone the Tansive repository:**

```bash
git clone https://github.com/tansive/tansive.git

cd tansive
```

#### **2. Start Tansive**

Tansive Server uses PostgreSQL as the datastore.  
 We provide a one-line setup that brings up:

- PostgreSQL
- Tansive Server
- The Tansive Runtime Agent (called **Tangent**)

```bash
docker compose -f scripts/docker/docker-compose-aio.yaml up -d
```

Wait for the _tangent_ container to start before proceeding.

#### **3. Install the Tansive CLI**

Choose your platform below for installation instructions:

<details id="linux">
<summary><strong>Linux</strong></summary>

Download and install the Tansive CLI for Linux:

**For x86_64/amd64 systems:**

```bash
# Download the latest release
curl -LO https://github.com/tansive/tansive/releases/download/v0.1.0-alpha.3/tansive-0.1.0-alpha.3-linux-amd64

# Make the binary executable
chmod +x tansive-0.1.0-alpha.3-linux-amd64

# Move the binary to a directory in your PATH
sudo install -m 0755 tansive-0.1.0-alpha.3-linux-amd64 /usr/local/bin/tansive

# Clean up
rm tansive-0.1.0-alpha.3-linux-amd64

# Verify installation
tansive version
```

**For arm64 systems:**

```bash
# Download the latest release
curl -LO https://github.com/tansive/tansive/releases/download/v0.1.0-alpha.3/tansive-0.1.0-alpha.3-linux-arm64

# Make the binary executable
chmod +x tansive-0.1.0-alpha.3-linux-arm64

# Move the binary to a directory in your PATH
sudo install -m 0755 tansive-0.1.0-alpha.3-linux-arm64 /usr/local/bin/tansive

# Clean up
rm tansive-0.1.0-alpha.3-linux-arm64

# Verify installation
tansive version
```

</details>

<details>
<summary><strong>macOS</strong></summary>

Download and install the Tansive CLI for macOS:

**For x86_64 / Intel Macs:**

```bash
# Download the latest release
curl -LO https://github.com/tansive/tansive/releases/download/v0.1.0-alpha.3/tansive-0.1.0-alpha.3-darwin-amd64

# Make the binary executable
chmod +x tansive-0.1.0-alpha.3-darwin-amd64

# Move the binary to a directory in your PATH
sudo install -m 0755 tansive-0.1.0-alpha.3-darwin-amd64 /usr/local/bin/tansive

# Clean up
rm tansive-0.1.0-alpha.3-darwin-amd64

# Verify installation
tansive version
```

**For Apple Silicon (M1/M2/M3) Macs:**

```bash
# Download the latest release
curl -LO https://github.com/tansive/tansive/releases/download/v0.1.0-alpha.3/tansive-0.1.0-alpha.3-darwin-arm64

# Make the binary executable
chmod +x tansive-0.1.0-alpha.3-darwin-arm64

# Move the binary to a directory in your PATH
sudo install -m 0755 tansive-0.1.0-alpha.3-darwin-arm64 /usr/local/bin/tansive

# Clean up
rm tansive-0.1.0-alpha.3-darwin-arm64

# Verify installation
tansive version
```

</details>

<details>
<summary><strong>Windows</strong></summary>

Download and install the Tansive CLI for Windows:

**Using Git Bash:**

```bash
# Create bin directory if needed
mkdir -p ~/bin

# Download the binary
curl -LO https://github.com/tansive/tansive/releases/download/v0.1.0-alpha.3/tansive-0.1.0-alpha.3-windows-amd64.exe

# Move to bin directory and rename
mv tansive-0.1.0-alpha.3-windows-amd64.exe ~/bin/tansive.exe

# Add to PATH for current session
export PATH="$HOME/bin:$PATH"

# Add to PATH permanently (add this line to your ~/.bashrc)
echo 'export PATH="$HOME/bin:$PATH"' >> ~/.bashrc

# Verify installation
tansive version
```

**Using WSL2:**

WSL2 provides a Linux environment on Windows. Follow the [Linux installation instructions](#linux) above, as WSL2 is fully compatible with the Linux x86_64 binary.

</details>

#### **4. Configure the CLI**

After installation, you'll need to configure the CLI to connect to your Tansive server.  
In this version Tansive server operates in single user mode. So you can login directly.

```bash
# Set the server URL (default exposed by the docker install)
tansive config --server https://local.tansive.dev:8678

# Login
tansive login

# Verify Status
tansive status
```

### Setup a Catalog

🕒 Total time: 3 minutes

**1. Create a .env file**

Create a `.env` file in your project root. Don't replace anything yet, we'll do it in the next steps.

```bash
# Create the .env file
# Replace only the API keys you want to use - keep placeholders for unused keys
cat > .env << 'EOF'
CLAUDE_API_KEY="<your-claude-key-here>"
OPENAI_API_KEY="<your-openai-key-here>"
# don't modify this. you don't need a kubernetes cluster!
KUBECONFIG="YXBpVmVyc2lvbjogdjEKa2luZDogQ29uZmlnCmNsdXN0ZXJzOgogIC0gbmFtZTogbXktY2x1c3RlcgogICAgY2x1c3RlcjoKICAgICAgc2VydmVyOiBodHRwczovL2Rldi1lbnYuZXhhbXBsZS5jb20KICAgICAgY2VydGlmaWNhdG9yaXR5LWRhdGE6IDxiYXNlNjQtZW5jb2RlZC1jYS1jZXJ0Pg=="
GITHUB_TOKEN="your-token-here"
GITHUB_CMD="/var/tangent/custom_scripts/github-mcp-server"
EOF
```

**2. Setup a Catalog**

Run the setup shell script that uploads a set of `yaml` files to Tansive.

```bash
# sets up a new catalog with variants called dev and prod, and SkillSets
bash examples/catalog_setup/setup.sh

# view the catalog structure that was setup
tansive tree
```

The Catalog is Tansive’s declarative inventory of environments, agents, and tools. Each variant (e.g., dev, prod) can contain SkillSets — collections of tools, agents, and workflows.

The setup script creates a hierarchical Catalog of resources in Tansive and configures policies to enable the examples in the next section.
The `tansive tree` command should show an output similar to:

```
myenv ❯ tansive tree
📁 Catalog
├── 🧬 default
├── 🧬 dev
│   └── 🌐 default
│       └── 🧠 SkillSets
│           ├── demo-skillsets
│           │   ├── health-record-demo
│           │   └── kubernetes-demo
│           └── secure-mcp-servers
│               └── github-mcp
└── 🧬 prod
    └── 🌐 default
        └── 🧠 SkillSets
            └── demo-skillsets
                └── kubernetes-demo
```

### Run the examples

#### Set up a policy-governed MCP proxy with input filtering for Github (or your favorite MCP server)

In this example, we will use Tansive to create a policy-governed tool proxy for popular MCP servers such as Github and configure with popular IDEs/utilities like Cursor, Windsurf, Copilot, Claude Desktop, etc. so you can protect against inappropriate or unintended agent actions or agents exposed to prompt injection risks.

Refer to [this article](https://docs.github.io/blog/implementing-defense-prompt-injection-attacks-mcp) to learn more about new security risks presented by LLM-backed agents and how Tansive provides the tools to enable effective defense.

<details><summary>Click to expand instructions</summary>

In the current version, Tansive supports running MCP Servers with `stdio` transport. Download the latest version of Github MCP Server from the [official release package](https://github.com/github/github-mcp-server/releases). Pick the latest version. Tansive is tested with 0.7.0 and should work with more latest versions as long as there are no tool name changes. Run the following commands from the project root.

```bash
mkdir -p custom_scripts
cd custom_scripts
# Choose the appropriate OS and arch compatible with your system
curl -LO https://github.com/github/github-mcp-server/releases/download/v0.7.0/github-mcp-server_Linux_x86_64.tar.gz
tar -xvf github-mcp-server_Linux_x86_64.tar.gz
```

> Note: Tansive can run docker distributions. However, in this setup we are running Tansive itself in docker, and therefore running an MCP server packaged in docker would involve DIND or mounting the docker UDS.

Open the `.env` file in your favorite editor and replace the placeholder value GITHUB_TOKEN with your actual access token created from your Github personal settings.

```bash
# Run the setup script again to update the resources
bash examples/catalog_setup/setup.sh

```

The setup scripts already created job function based View policies like developer, tester, devops-engineer etc.  
Create a policy-filtered MCP endpoint as below:

```bash
tansive session create /secure-mcp-servers/github-mcp/github-mcp-server --view developer
```

You can run multiple concurrent sessions with different profiles and add them to your IDE like cursor and switch between them.

Read the [documentation for the Github MCP example](examples/README.md) for how the roles are configured and what capabilities they enable. You can modify the example to suit your needs or use it as a template for configuring another MCP server.

</details>

#### Run the Policy-driven agent

In this example, you'll simulate an agent solving a Kubernetes incident. You'll create a catalog, define tools and policies, and watch how Tansive enforces them differently across environments - without needing a real cluster.

<details><summary>Click to expand instructions</summary>

Before running this example, you have to configure an API Key for either OpenAI or Anthropic. Open the .env file and replace the placeholder with the API Key.

**Scenario**

This is a fictional debugging scenario involving an e-commerce application deployed on Kubernetes. The application is unable to take orders, and you'll use an AI Agent to investigate the issue.

Two tools are available to the agent:

**`list-pods`** - lists the status of running pods using a label selector.

**`restart-deployment`** - restarts a deployment by name.

Both tools are shell scripts that return mock output formatted like `kubectl`, so you can run the example without needing a real Kubernetes cluster. The tools are implemented in `skillset_scripts/tools_script.sh`

The purpose of this example is to show how Tansive enforces policy at runtime. Specifically, we'll **block** the use of `restart-deployment` in the _prod_ environment, but **allow** it in _dev_ environment.

> In Tansive, agents and tools are called **Skills** and a collection of Skills along with their run time context is called a **SkillSet**

**Run the Skill**

Start a session in the **dev** environment using the `k8s_troubleshooter` Skill:

> **Note for Windows Git Bash:**
> Add a `/` in front of the skill path. For example:
> `//demo-skillsets/kubernetes-demo/k8s_troubleshooter`

If you provided a CLAUDE-API-KEY, change the `"model"` value to `"claude"`

```bash
tansive session create /demo-skillsets/kubernetes-demo/k8s_troubleshooter \
--view dev-view \
--input-args '{"prompt":"An order-placement issue is affecting our e-commerce system. Use the provided tools to identify the root cause and take any necessary steps to resolve it.","model":"gpt4o"}' \
--interactive
```

Run the same session in the **prod** environment to see how Tansive blocks `restart-deployment`. Replace `dev-view` with `prod-view` in the above example.

</details>

#### Run the Secure Data Handling example

In this hands-on example, you'll simulate a health bot that answers medical questions while enforcing strict access controls. You'll see how Tansive can protect Personal Health Information (PHI) in real time.

<details><summary>Click to expand instructions</summary>

**Scenario**

This is a fictional debugging scenario involving a health bot that answers questions about an authenticated caller's health.

Two tools are available to the agent:

**`resolve-patient-id`** - provides the ID of a patient (`patient_id`), given their name. This tool is written in Javascript `skillset_scripts/resolve-patient-id.js`

**`patient-bloodwork`** - returns patient's blood test results, given their `patient_id`. This tool is written in Python `skillset_scripts/patient_bloodwork.py`

The purpose of this example is to show how Tansive can be used to validate and filter inputs to enforce data boundaries. Specifically, you'll **pin the session** to John's `patient_id` so that any attempt to access records for other patients, like Sheila, will be blocked automatically.

**Run the Skill**

Start a session in the **dev** environment using the `health-record-agent` Skill:

> **Note for Windows Git Bash:**
> Add a `/` in front of the skill path.
> For example: `//demo-skillsets/health-record-demo/health-record-agent`

If you provided a CLAUDE-API-KEY, change the `"model"` value to `"claude"`

```bash
tansive session create /demo-skillsets/health-record-demo/health-record-agent \
--view dev-view --input-args '{"prompt":"John Doe and Sheila Smith were looking sick. Can you please check their bloodwork and tell me if theres anything wrong?","model":"claude"}' \
--session-vars '{"patient_id":"H12345"}'
```

</details>

### Explore the Audit Log

Now you’ll retrieve the audit logs for your example runs.

Audit logs are different from debug logs, which are typically printed to the console or sent to external systems for indexing and search. An audit log in Tansive is an immutable, tamper-evident record of the steps and events that occurred during a session. Tansive logs audit events separately from debug logs. You can learn more about the structure and verification of audit logs in the [concepts](/docs/concepts.md) section of the docs.

Get the list of sessions:

```bash
tansive session list
```

Copy a SESSION ID and paste it into the placeholder below:

```bash
tansive session audit-log get your-session-id-here -o friendly_name.tlog
```

Verify and open the log:

```bash
# Verify the log's integrity
tansive session audit-log verify friendly_name.tlog

# Generate and open a user-friendly HTML view
tansive session audit-log view friendly_name.tlog
```

The `verify` command validates the hash chain to confirm the log has not been tampered with. The `view` command generates an HTML version of the log. You can pass `--no-open` to `view` if you prefer to create the HTML file without automatically launching your browser.

> **Windows Users:** If you are using Git Bash or WSL2, the browser may not launch automatically. A `.html` file will be generated in the same directory, which you can manually double-click in Windows Explorer to open in your browser.

[View sample audit log](https://docs.tansive.io/samples/friendly_name.html) generated from the Kubernetes example. This log was produced using Claude, which called `list-pods` multiple times to confirm that the deployment restarted successfully. Skill invocations are sorted by timestamp and nested to show which calls invoked others.

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
architecture, I purposely built Tansive to be easily extensible. Trust and extensibility don't work behind closed doors.

#### Why should we trust this project now?

Tansive is in early Alpha, and it's not ready for production use. But the foundations - hierarchical organization of agent and tool assets, policy-based views, dynamic runtime control via transforms, language and framework agnostic runtime, tamper-evident logs, and extensible SkillSet abstractions - are designed to enable and sustain wide adoption of agents to automate day to day tasks without compromising on security and compliance.

I hope you will try Tansive in your non-production environments with real workloads and provide feedback on the problems you face and the capabilities you’d like Tansive to deliver. Your insights will help shape a platform that aspires to become the standard for secure, agent-driven workflows. Thank you in advance for being part of this journey.

#### I see a large initial commit. Where is this coming from?

Tansive was developed privately by a single author [@anand-tan](https://github.com/anand-tan), and then moved to this repository to provide a clean starting point for open-source development. The [repositories](https://github.com/anand-tnsv) are publicly archived for historical reference.

## 🛠️ Dependencies:

Tansive builds on widely adopted, well-tested open-source components, including:

- Go standard library
- PostgreSQL (for catalog storage)
- Common libraries for YAML parsing, HTTP handling, and CLI UX
- No custom cryptography

Additional dependencies are listed in [`go.mod`](./go.mod)

📄 [Concepts](/docs/concepts.md)

## 🛡️ Security Notice:

Tansive is in early alpha. While built on established components, it has not undergone third-party security audits.
Use with caution in sensitive or production environments.

Refer [SECURITY.md](/SECURITY.md)
