# Concepts

## Overview

This section introduces the key concepts and components behind Tansive. We'll refer to the example from the [Getting Started](/README.md#-getting-started) guide to illustrate how these ideas come together in practice.

At a high level, Tansive is composed of:

- **Skills:** An Agent or a tool - a runnable unit of execution. Each Skill accepts a structured input, provides a structured output, and implements a named capability.
- **SkillSets:** Bundles that group related Skills along with their capabilities, implementation, and contextual data. SkillSets are declaratively specified.
- **Resources:** Data sources such as databases, external APIs, secrets, or configuration values. Resources can be read from or written to by Skills, and are primarily modeled using CRUD primitives. Resources may be global (shared across SkillSets) or session-scoped (isolated to a specific execution of a Skill).
- **Catalog:** The central repository of all SkillSets and Resources. It supports a hierarchical, path-based structure, enabling easy referencing.
- **Views:** Filtered projections of the Catalog that define what parts are visible and what actions are allowed. Access to a View is granted via an access token scoped to the View.
- **Tansive Server:** The control plane of Tansive. It hosts the Catalog, enforces policies, manages configuration, and co-ordinates the invocation and execution of Skills.
- **Tangent:** The runtime agent that executes Skills. (No, this one doesn't go off-topic!) Tangent enforces runtime policies, mediates skill-to-skill calls, and governs access to both global and session-scoped Resources. Each Tangent runs one or more Runners, which are responsible for executing individual Skills. Multiple Tangents can be connected to a single Tansive Server.

In the following sections, we’ll dive into Skills, SkillSets, Resources, Catalog, and Views, with examples, use cases, and how they fit into the Tansive workflow. Then we'll dive in to the Architecture and discuss Tansive Server and Tangent.

## Core Concepts

Let’s walk through what happens when you run a Skill in the example: You start from the CLI, invoke a Skill from a SkillSet, which is executed by Tangent under a scoped View. The Skill reads context values such as API Keys, makes calls to other Skills which are in turn executed by Tangent, and finally returns results back to the CLI.

### Skill

A Skill in Tansive is a portable unit of execution. In simple terms, it's a callable function. Its job is simple: take structured input, perform some operation, and return structured output. Skills are powerful in terms of their flexibility - they can be anything from a basic shell utility to a complex agent powered by an LLM. They can be written in any language, and they all follow the same interface contract.

**Sample Script**

Let’s look at a concrete example from the example. The `list_pods` and `restart_deployment` Skills are implemented in a Bash script. Both are handled in the same script by sending a mock response based on the `skillName` field in the input.

```bash
set -e

# Read entire input
INPUT=$1

# Extract fields from JSON
SKILL_NAME=$(echo "$INPUT" | jq -r '.skillName')
INPUT_ARGS=$(echo "$INPUT" | jq -c '.inputArgs')

# Respond based on skill name
case "$SKILL_NAME" in
  list_pods)
    LABEL_SELECTOR=$(echo "$INPUT_ARGS" | jq -r '.labelSelector')
    echo "NAME                                READY   STATUS    RESTARTS   AGE"
    echo "api-server-5f5b7f77b7-zx9qs          1/1     Running   0          2d"
    echo "web-frontend-6f6f9d7b7b-xv2mn        1/1     Running   1          5h"
    echo "cache-7d7d9d9b7b-pv9lk        1/1     Running   0          1d"
    echo "orders-api-7ff9d44db7-abcde          0/1     CrashLoopBackOff   12         3h"
    echo "# Filter applied: $LABEL_SELECTOR"
    ;;
  restart_deployment)
    DEPLOYMENT=$(echo "$INPUT_ARGS" | jq -r '.deployment')
    echo "deployment.apps/$DEPLOYMENT restarted"
    ;;
  *)
    echo "Unknown skillName: $SKILL_NAME" >&2
    exit 1
    ;;
esac

```

This script reads a JSON-encoded argument (passed as a single string), extracts fields with `jq`, and runs the appropriate logic. In this case, `list_pods` simulates the output of a Kubernetes `kubectl get pods` command, optionally filtered by label provided as input. The `restart_deployment` Skill simulates a deployment restart with the name of the deployment provided as input. Each Skill can inspect `inputArgs`, perform the task, and print a response.

By convention, Tansive passes a single JSON blob as input to each Skill. That input includes:

    - **`skillName`:** the name of the function being invoked.
    - **`inputArgs`:** the structured arguments passed by the caller.
    - **`sessionVars`:** session scoped values that are pinned to a session.

This approach allows multiple Skills to be implemented in the same script or binary. This simplifies dispatch logic and works across languages, from Bash to Python, Node.js, compiled Go, or anything else. Importantly, even when multiple Skills are bundled in a single executable, Tansive can enforce distinct access policies for each Skill individually. This ensures flexibility in implementation without compromising security or policy enforcement.

**The takeaway:** if you can write a function in any language that takes input and returns output, you can turn it into a Skill.

**Declarative YAML definition**

A Skill is defined declaratively within a SkillSet bundle, as shown below:

```yaml
- name: restart_deployment
  source: my-tools-script
  description: "Restart a deployment"
  inputSchema:
    type: object
    properties:
      deployment:
        type: string
    required:
      - deployment
  outputSchema:
    type: string
    description: "Raw output from restarting the deployment, typically from 'kubectl rollout restart deployment <deployment>'"
  exportedActions:
    - kubernetes.deployments.restart
  annotations:
    llm:description: |
      Performs a rollout restart of a Kubernetes deployment.
      This Skill is used to trigger a fresh rollout of pods associated with a deployment,
      typically to recover from failures or apply configuration changes.
      It requires the deployment name as input and will execute the equivalent of kubectl rollout restart deployment <name>.
```

The YAML includes:

- **name**: The name of the Skill, which will be passed in `skillName`.
- **source**: Pointer to the script or binary that implements the Skill logic. This will be explained in the following section on SkillSets.
- **description**: Human readable description of what the skill does.
- **inputSchema** and **outputSchema**: JSON Schema definitions that describe the expected input and output. This serves two purposes: (1) They help agents understand how to call the Skill correctly (2) Tansive validates all inputs against the schema at runtime.
- **exported actions**: A list of named capabilities that this Skill provides. These are used in Tansive's View policy definitions to control access. In this example, the Skill exports `kubernetes.deployments.restart`.
- **annotations**: These are optional metadata. For example, Skills with an `llm:description` gives AI Agents a description of what the skill does. Skills without this annotation will not be available as tools for LLM-based agents.

Together, this structure gives Tansive a way to validate input, enforce policy, and make Skills discoverable and composable.

Irrespective of whether the Skill implements a local computation, accesses an external service, or implements an autonomous agent backed by an LLM, the Skill abstraction keeps the interface consistent.

**Input Transforms and Session Pinning**

Skills can also be defined with a **transform function** that is declaratively specified in the YAML configuration.

A **transform** is a JavaScript function that runs on the input **before** the Skill is invoked. It receives:

- The **pinned session variables**, and
- The **input payload** provided by the caller.

The transform function can:

- **Throw an error**  
  If the input fails to meet certain criteria or does not match values pinned to the session.

- **Return the input without modification**  
  If no changes are needed, simply return the input as-is, and it will be passed directly to the Skill.

- **Modify the input or return a new value**  
  The transform can produce any value that complies with the Skill’s `inputSchema`.

In the **Health-bot** example, we defined a transform function to validate the `patient_id` against the value pinned to the session.

```yaml
- name: patient-bloodwork
      source: patient-bloodwork
      description: "Get patient bloodwork"
      inputSchema:
        type: object
        properties:
          patient_id:
            type: string
            description: "Patient ID"
        required:
          - patient_id
      outputSchema:
        type: object
        description: "Raw output from patient bloodwork in json"
      transform: |
        function(session, input) {
          if (session.patient_id != input.patient_id) {
            throw new Error('Unauthorized to access patient bloodwork for patient ' + input.patient_id)
          }
          return {
            patient_id: input.patient_id
          }
        }
      exportedActions:
        - patient.labresults.get
      annotations:
        llm:description: |
          Get patient bloodwork from the patient's health record.
          This skill is used to retrieve the patient's bloodwork from the patient's health record.
          It requires the patient ID as input and will return the patient's bloodwork in json.

```

In this example, the transform function compared the `patient_id` provided as input against the value pinned to the session. If they matched, it returned the `patient_id` unmodified; otherwise, it threw an error.

Transforms can be useful in many ways:

- **Validate sensitive data** such as PHI, PII, or PCI data.
- **Morph inputs** to match expected values from the script or combine them with values pinned to the session. For example, the Kubernetes example could have pinned a label that a transform uses to decorate the input.
- **Apply feature flags** to make tools and agents behave differently based on the session context.

**The Takeaway:** Declarative policies enforce static contracts, while transform functions provide dynamic, context-driven safeguards.

> **Why call them Skills?**
>
> While _Tools_ and _Agents_ are common terms, Tansive treats both as runnable units of execution that implement _capabilities_. A common abstraction enables flexible chaining and uniform enforcement of policy.

### SkillSet

A **SkillSet** is a package of related skills that are designed to work together. It defines both the Skills (individual functions or agents) and the contextual data (called _Contexts_) they share at runtime.

A SkillSet is analogous to a `class` in object-oriented programming.

- Skills are like `methods` of the class - discrete, callable units of logic
- Contexts are like instance variables - shared data available to all Skills during execution.

Like a `class`, a SkillSet definition is a template. It’s instantiated in a runtime session by the Tangent, which loads the actual implementations of Skills and executes them within the context of the session.

Skills can range from simple tools (e.g., shell scripts, API calls) to LLM-backed autonomous agents. Tansive ensures they can collaborate securely within the bounds of a session.

In this section we'll dive in to how to define a SkillSet. How the SkillSet is instantiated and executed at runtime will be covered by the section on Tangent.

A SkillSet definition has three distinct parts: `skills`, `context`, and `sources`

The general structure of a SkillSet definition is as below:

```yaml
apiVersion: 0.1.0-alpha.1
kind: SkillSet
metadata:
  name: kubernetes-demo # unique name for this skillset
  path: /skillsets # hierarchical path in the catalog where this sits
spec:
  version: "0.1.0" # Tansive SkillSet spec schema version
  sources:
    - name: my-agent-source
    # source definition
    - name: my-tool-source
    # source definition
  context:
    - name: a-context-for-my-skills
    # context definition
    - name: another-context
    # context definition
  skills:
    - name: list-pods
    # skill definition
    - name: restart-deployment
    # skill definition
```

**Skills**

We already looked at Skill definitions in the previous section. The `skills` section is an array of individual Skill definitions. Each Skill must reference a `source` defined in the `sources` section. This linkage tells Tangent how to locate and execute the Skill during a session.

Tansive enables a clean separation between defining what a Skill does (defined in `skills`) and how it runs (defined in `sources`), enabling flexible composition and secure execution.

```yaml
skills:
  - name: restart_deployment
    source: my-tools-script
    description: "Restart a deployment"
    inputSchema:
      type: object
      properties:
        deployment:
          type: string
      required:
        - deployment
    outputSchema:
      type: string
      description: "Raw output from restarting the deployment, typically from 'kubectl rollout restart deployment <deployment>'"
    exportedActions:
      - kubernetes.deployments.restart
    annotations:
      llm:description: |
        Performs a rollout restart of a Kubernetes deployment.
        ...
  - name: k8s_troubleshooter
    source: my-agent
    description: "Troubleshoot kubernetes issues"
    inputSchema:
      type: object
      properties:
        prompt:
          type: string
          description: "Description of the issue to troubleshoot"
      required:
        - prompt
    outputSchema:
      type: string
      description: "Troubleshooting steps and recommendations"
    exportedActions:
      - kubernetes.troubleshoot
    annotations:
      llmx:description: |
        A Kubernetes troubleshooting assistant that helps diagnose and resolve issues in your cluster.
        ...
```

**Context**

Context represents shared runtime state available to all Skills in a SkillSet. It allows Skills to read configuration values, pass data, cache results, or reference external inputs during execution.

Here’s an example of a Context definition:

```yaml
context:
  - name: kubeconfig
    schema:
      type: object
      properties:
        kubeconfig:
          type: string
          format: binary
      required:
        - kubeconfig
    value:
      kubeconfig: "....."
```

In this example, the context provides a kubeconfig value. The schema enforces that this value is a required binary-formatted string.

In the current release of Tansive, only JSON object contexts are supported. This is sufficient for most automation tasks. Upcoming releases will prioritize support for additional context types, including secrets, in-memory vector stores, and further expanding to external stores for session-scoped caching like Redis.

:::info Storing sensitive values
Sensitive values like kubeconfigs and API keys should not be stored directly inside JSON blobs. These will be migrated to `secret`-typed contexts once supported.
:::

**Sources**

Sources tell the Tangent how to run the Skills defined in a SkillSet. We’ll cover Sources and Runners in more detail in the Tangent section, but for completeness, here’s an example _source_ definition:

```yaml
sources:
  - name: my-agent
    runner: "system.stdiorunner" #type of runner
    config:
      version: "0.1.0-alpha.1" # version of the source definition schema
      runtime: "python" # type of runtime: python, node, bash, binary
      env:
        TEST_VAR: "test_value" # environment variables available during execution
      script: "run-llm.py" # name of script or executable
      security:
        type: dev-mode # could be one of: dev-mode, sandboxed
  - name: my-tools-script
    runner: "system.stdiorunner"
    config:
      version: "0.1.0-alpha.1"
      runtime: "bash"
      env:
        TEST_VAR: "test_value"
      script: "tools_script.sh"
      security:
        type: dev-mode # could be one of: dev-mode, sandboxed
```

A Source has three key parts:

- **name:** A unique name used to reference this source from within Skills. A single source can expose multiple Skills.
- **runner:** The runner responsible for executing the source. Tansive currently supports `system.stdiorunner`, which runs local scripts and returns output from `stdout` and `stderr`. Input to the Skill is passed via JSON-encoded arguments. Future releases will support runners that invoke remote APIs, launch serverless functions, or even interact with long-running services.
- **config:** Runner-specific configuration. For `system.stdiorunner`, this includes the runtime (python, node, bash, or binary), environment variables, script path, and a security mode (dev-mode or sandboxed).

> For a complete example of a SkillSet definition, refer to `catalog_config/skillset-k8s.yaml` in the cloned installation repository.

### Resources

Resources are shared, persistent entities accessible across SkillSets. While SkillSets are like classes in object-oriented programming, Resources are more like global variables and are common across sessions.

A Resource might represent:

- A secret value from a vault
- A row in a relational table fetched via SQL
- A document in a key-value store
- A remote REST endpoint

Tansive models Resources using standard CRUD semantics, allowing Skills to read from, write to, and update external systems securely and declaratively.

> Support for Resources is actively evolving and will be available in the 0.1.0 major release. We’re learning from early adopters to prioritize support for the most useful resource types first.

### Catalog

Catalog is the repository that holds all the Resource and SkillSet definitions. These definitions are arranged hierarchically in a path-like structure similar to directories in a file-system.

For example:

```bash
/skillsets/devops-tools/kubernetes-troubleshooter
/resources/access-tokens/zendesk-token
```

**Namespaces**

The Catalog can be organized in to **Namespaces** which provide a way to define an isolated hierarchy of Resource and SkillSets. Each team or function can operate within their own Namespace giving them a dedicated, isolated space to define and manage the automation components relevant to their function. Within a Namespace, teams can organize their content hierarchically in the same path-like structure.

```base
📁 catalog
├── 🌐 payments-team-namespace
│   ├── 📁 resources
│   │   ├── db-creds
│   │   └── stripe-key
│   └── 📁 skillsets
│       ├── 📁 payment-processing
│       │   └── charge-user
│       └── 📁 reconciliation
│           └── reconcile-ledger
│
├── 🌐 support-team-namespace
│   ├── 📁 resources
│   │   └── zendesk-token
│   └── 📁 skillsets
│       ├── 📁 ticketing
│       │   └── auto-triage
│       └── 📁 escalation
│           └── escalate-to-oncall
│
├── 🌐 devops-team-namespace
│   ├── 📁 resources
│   │   └── kubeconfig
│   └── 📁 skillsets
│       ├── 📁 cluster-management
│       │   └── restart-deployment
│       └── 📁 monitoring
│           └── diagnose-pod
│
├── 🌐 marketing-team-namespace
│   ├── 📁 resources
│   │   └── hubspot-api
│   └── 📁 skillsets
│       ├── 📁 campaigns
│       │   └── launch-campaign
│       └── 📁 analytics
│           └── generate-leads-report

```

In Tansive, Views can be scoped to a single Namespace hence providing soft-multitenancy within the Catalog.

**Variants**

Tansive supports Variants of the entire Catalog — such as dev, stage, and prod — enabling multiple replicas of the system’s structure, each with its own runtime values. This allows teams to test, iterate, and deploy changes safely across environments while re-using the same logical definitions.

> It is not necessary to preserve the same structure or even the same namespaces or components across the variants. A Variant is simply another container within the Catalog.

The following diagram illustrates how the Catalog is organized with Variants and Namespaces:

Fig. 2 - Catalog Structure

<div style={{textAlign: 'center', border: '1px solid #444', padding: '10px', marginBottom: '20px', borderRadius: '8px'}}>

![Catalog Structure](./assets/catalog-structure.svg)

</div>

**Declarative Definitions**

The entire Catalog and its contents and structure can be declaratively defined in YAML and version controlled in Git, allowing for easy integration with existing CI/CD workflows. You can create and update the Catalog and its contents and structure via the `tansive` CLI.

In the example, we created a catalog using the following YAML:

```yaml
apiVersion: "0.1.0-alpha.1"
kind: Catalog
metadata:
  name: demo-catalog # A unique name for the Catalog in your Tansive installation
  description: "This is the catalog for a demo of Tansive" # A human-readable description
```

We then created `dev` and `prod` variants, and a namespace within the `dev` variant.

```yaml
apiVersion: "0.1.0-alpha.1"
kind: Variant
metadata:
  name: dev # a unique name for the Variant within the Catalog
  catalog: demo-catalog
  description: "Variant for the development environment"
---
apiVersion: "0.1.0-alpha.1"
kind: Variant
metadata:
  name: prod # a unique name for the Variant within the Catalog
  catalog: demo-catalog
  description: "Variant for the production environment"
---
apiVersion: "0.1.0-alpha.1"
kind: Namespace
metadata:
  name: my-app-ns # a unique name for the Namespace within the Variant
  variant: dev
  description: "An example namespace"
```

Finally, we created the same SkillSet definition in both `dev` and `prod` variants using:

```bash
tansive create -f catalog_config/skillset-k8s.yaml --variant dev
tansive create -f catalog_config/skillset-k8s.yaml --variant prod
```

The Catalog we created for the example can be visualized by running the following command:

```bash
tansive tree
```

It should show an output similar to:

```
myenv ❯ tansive tree
📁 Catalog
├── 🧬 default
├── 🧬 dev
│   └── 🌐 default
│       └── 🧠 SkillSets
│           └── demo-skillsets
│               ├── kubernetes-demo
│               └── health-record-demo
└── 🧬 prod
    └── 🌐 default
        └── 🧠 SkillSets
            └── demo-skillsets
                ├── kubernetes-demo
                └── health-record-demo
```

### Views

Views are filtered projections of the Catalog based on a set of rules. In Tansive, all policies are defined and enforced through Views. Let's look at an example: the `dev-view` used in the Kubernetes example.

```yaml
apiVersion: 0.1.0-alpha.1
kind: View
metadata:
  name: dev-view
  catalog: demo-catalog
  variant: dev
  description: View with full access to resources
spec:
  rules:
    - intent: Allow
      actions:
        - system.skillset.use
        - kubernetes.pods.list
        - kubernetes.deployments.restart
        - kubernetes.troubleshoot
      targets:
        - res://skillsets/devops_skillsets/kubernetes-demo
    - intent: Allow
      actions:
        - system.skillset.use
        - patient.labresults.get
        - patient.id.resolve
      targets:
        - res://skillsets/clinical_skillsets/health-record-demo
```

A View consists of two parts: Scope and Rules.

**Scope** The scope for a view is defined by its metadata. In this example, the View is scoped to the Catalog `demo-catalog` and the Variant `dev`. This means it cannot access any components in other Variants or Catalogs. Scope acts as a coarse-grained access boundary.

**Rules** Rules specify the fine-grained actions permitted within the View. These include both system-defined actions (such as `system.skillset.use`) and capabilities exported by SkillSets (like `kubernetes.pods.list`). In this case, the View grants permission to invoke the `kubernetes-demo` SkillSet and allows access to specific Kubernetes-related actions. It also allows access to certain actions in the `health-record-demo` SkillSet under the `clinical_skillsets` path.

A separate page on Views covers system actions in more detail and outlines the different ways Views can be defined and composed.
