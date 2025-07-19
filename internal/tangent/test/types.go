package test

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/tansive/tansive/internal/catalogsrv/policy"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"gopkg.in/yaml.v3"
)

const skillsetDef = `
apiVersion: 0.1.0-alpha.1
kind: SkillSet
metadata:
  name: kubernetes-demo
  catalog: test-catalog
  variant: test-variant
  path: /skillsets
spec:
  version: "0.1.0"
  sources:
    - name: my-agent-script
      runner: "system.stdiorunner"
      config:
        version: "0.1.0-alpha.1"
        runtime: "bash"
        runtimeConfig:
          key: "value"
        env:
          TEST_VAR: "test_value"
        script: "test_script.sh"
        security:
          type: default  # could be one of: default, sandboxed
    - name: my-tools-script
      runner: "system.stdiorunner"
      config:
        version: "0.1.0-alpha.1"
        runtime: "bash"
        runtimeConfig:
          key: "value"
        env:
          TEST_VAR: "test_value"
        script: "tools_script.sh"
        security:
          type: default  # could be one of: default, sandboxed
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
        kubeconfig: YXBpVmVyc2lvbjogdjEKa2luZDogQ29uZmlnCmNsdXN0ZXJzOgogIC0gbmFtZTogbXktY2x1c3RlcgogICAgY2x1c3RlcjoKICAgICAgc2VydmVyOiBodHRwczovL2Rldi1lbnYuZXhhbXBsZS5jb20KICAgICAgY2VydGlmaWNhdGUtYXV0aG9yaXR5LWRhdGE6IDxiYXNlNjQtZW5jb2RlZC1jYS1jZXJ0Pg==
      annotations: {}
  skills:
    - name: list_pods
      source: my-tools-script
      description: "List pods in the cluster"
      inputSchema:
        type: object
        properties:
          labelSelector:
            type: string
            description: "Kubernetes label selector to filter pods"
        required: []
      outputSchema:
        type: string
        description: "Raw output from listing pods, typically from 'kubectl get pods'"
      exportedActions:
        - kubernetes.pods.list
      annotations:
        llm:description: |
          Lists all pods in the currently active Kubernetes cluster. This skill supports an optional labelSelector argument to filter pods by label. It is a read-only operation that provides visibility into running or failing workloads. The output is a plain-text summary similar to kubectl get pods. Use this to diagnose the current state of the system.
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
          Performs a rollout restart of a Kubernetes deployment. This skill is used to trigger a fresh rollout of pods associated with a deployment, typically to recover from failures or apply configuration changes. It requires the deployment name as input and will execute the equivalent of kubectl rollout restart deployment <name>.
    - name: k8s_troubleshooter
      source: my-agent-script
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
          A Kubernetes troubleshooting assistant that helps diagnose and resolve issues in your cluster. This skill accepts natural language descriptions of problems and leverages knowledge of Kubernetes concepts and common failure patterns to provide targeted diagnostic steps and remediation advice. It can help identify issues related to pod failures, networking problems, resource constraints, configuration errors and more. The assistant will analyze the situation and suggest appropriate kubectl commands and configuration changes to resolve the issue.
`

func SkillsetDef(env string) json.RawMessage {
	jsonData := getJsonFromYaml(skillsetDef)
	if env == "dev" {
		sjson.SetBytes(jsonData, "spec.context.0.value.kubeconfig", "YXBpVmVyc2lvbjogdjEKa2luZDogQ29uZmlnCmNsdXN0ZXJzOgogIC0gbmFtZTogbXktY2x1c3RlcgogICAgY2x1c3RlcjoKICAgICAgc2VydmVyOiBodHRwczovL2Rldi1lbnYuZXhhbXBsZS5jb20KICAgICAgY2VydGlmaWNhdGUtYXV0aG9yaXR5LWRhdGE6IDxiYXNlNjQtZW5jb2RlZC1jYS1jZXJ0Pg==")
	} else {
		sjson.SetBytes(jsonData, "spec.context.0.value.kubeconfig", "YXBpVmVyc2lvbjogdjEKa2luZDogQ29uZmlnCmNsdXN0ZXJzOgogIC0gbmFtZTogbXktY2x1c3RlcgogICAgY2x1c3RlcjoKICAgICAgc2VydmVyOiBodHRwczovL3Byb2QtZW52LmV4YW1wbGUuY29tCiAgICAgIGNlcnRpZmljYXRlLWF1dGhvcml0eS1kYXRhOiA8YmFzZTY0LWVuY29kZWQtY2EtY2VydD4=")
	}
	return jsonData
}

func getJsonFromYaml(yamlStr string) json.RawMessage {
	var yamlData interface{}
	if err := yaml.Unmarshal([]byte(yamlStr), &yamlData); err != nil {
		panic(err)
	}
	jsonData, err := json.Marshal(yamlData)
	if err != nil {
		panic(err)
	}
	return jsonData
}

func SkillsetPath() string {
	jsonData := getJsonFromYaml(skillsetDef)
	name := gjson.Get(string(jsonData), "metadata.name").String()
	path := gjson.Get(string(jsonData), "metadata.path").String()
	return fmt.Sprintf("%s/%s", path, name)
}

func SkillsetAgent() string {
	return "k8s_troubleshooter"
}

const devView = `
{
  "apiVersion": "0.1.0-alpha.1",
  "kind": "View",
  "metadata": {
    "name": "dev-view",
    "catalog": "test-catalog",
    "variant": "dev",
    "description": "View with full access to resources"
  },
  "spec": {
    "rules": [{
      "intent": "Allow",
      "actions": ["system.skillset.use","kubernetes.pods.list", "kubernetes.deployments.restart", "kubernetes.troubleshoot"],
      "targets": ["res://skillsets/skillsets/kubernetes-demo"]
    },
    {
      "intent": "Allow",
      "actions": ["system.skillset.use", "supabase.mcp.use", "supabase.tables.list", "supabase.sql.query", "supabase.sql.superuser"],
      "targets": ["res://skillsets/skillsets/supabase-demo"]
    }]
	}
}`

func GetView(name string) json.RawMessage {
	view, _ := sjson.Set(devView, "metadata.name", name)
	return json.RawMessage(view)
}

func GetViewDefinition(variant string) *policy.ViewDefinition {
	vd := policy.ViewDefinition{
		Scope: policy.Scope{
			Catalog: "test-catalog",
			Variant: variant,
		},
	}
	rules := []policy.Rule{}
	rulesJson := gjson.Get(devView, "spec.rules").Raw
	err := json.Unmarshal([]byte(rulesJson), &rules)
	if err != nil {
		panic(err)
	}
	if variant == "prod" {
		rules[0].Actions = []policy.Action{"system.skillset.use", "kubernetes.pods.list", "kubernetes.troubleshoot"}
	}
	vd.Rules = rules
	return &vd
}

const mcpSkillsetDef = `
apiVersion: 0.1.0-alpha.1
kind: SkillSet
metadata:
  name: supabase-demo
  catalog: test-catalog
  variant: test-variant
  path: /skillsets
spec:
  version: "0.1.0"
  sources:
    - name: supabase-mcp-server
      runner: system.mcp.stdio
      config:
        version: "0.1.0"
        command: npx
        args:
          - -y
          - "@supabase/mcp-server-supabase@latest"
          # project-ref will be set in code
        env:
          # SUPABASE_ACCESS_TOKEN will be set in code
    - name: sql-validator
      runner: system.stdiorunner
      config:
        version: "0.1.0-alpha.1"
        runtime: "python"
        script: "validate_sql.py"
        security:
          type: default
  context:
    - name: sql-permissions
      schema:
        type: object
        properties:
          allow:
            type: object
            additionalProperties:
              type: array
              items:
                type: string
          deny:
            type: object
            additionalProperties:
              type: array
              items:
                type: string
        required:
          - allow
          - deny
      value:
        allow:
          select:
            - support_tickets
            - support_messages
          update:
            - support_messages
        deny:
          all:
            - integration_tokens
      valueByAction:
        - action: supabase.sql.superuser
          value:
            allow:
              all:
                - support_tickets
                - support_messages
                - integration_tokens
        - action: supabase.sql.query
          value:
            deny:
              all:
                - integration_tokens
      attributes: 
        readOnly: true
        exportedActions:
          - supabase.sql.superuser
          - supabase.sql.query
  skills:
    - name: validate_sql
      source: sql-validator
      description: Validate SQL input
      inputSchema:
        type: object
        required:
          - sql
        properties:
          sql:
            type: string
      outputSchema:
        type: object
      exportedActions:
        - supabase.mcp.use
    - name: list_tables
      source: supabase-mcp-server
      exportedActions:
        - supabase.tables.list
    - name: execute_sql
      source: supabase-mcp-server
      description: Execute SQL query
      transform: |
        function(session, input) {
          let validationInput = {
            sql: input.query
          }
          let ret = SkillService.invokeSkill("validate_sql", validationInput);
          // if ret is not an object, throw an error
          if (typeof ret !== "object") {
            throw new Error("unable to validate input");
          }
          if(!ret.allowed) {
            throw new Error(ret.reason);
          }
          console.log("input validated");
          console.log(ret);
          return input;
        }
      exportedActions:
        - supabase.sql.query
    - name: supabase_mcp
      source: supabase-mcp-server
      description: Supabase MCP server
      exportedActions:
        - supabase.mcp.use
      annotations:
        mcp:tools: filter-tools
`

func getMCPSkillsetDef(env string) json.RawMessage {
	jsonData := getJsonFromYaml(mcpSkillsetDef)
	projectRef := os.Getenv("SUPABASE_PROJECT")
	accessToken := os.Getenv("SUPABASE_ACCESS_TOKEN")

	// Set the project-ref argument (assuming it's the 4th arg, index 3)
	if projectRef != "" {
		jsonData, _ = sjson.SetBytes(jsonData, "spec.sources.0.config.args.3", "--project-ref="+projectRef)
	}

	// Set the access token in env
	if accessToken != "" {
		jsonData, _ = sjson.SetBytes(jsonData, "spec.sources.0.config.env.SUPABASE_ACCESS_TOKEN", accessToken)
	}

	if env == "prod" {
		jsonData, _ = sjson.DeleteBytes(jsonData, "spec.skills.2.transform")
	}

	return jsonData
}

func MCPSkillsetPath() string {
	jsonData := getJsonFromYaml(mcpSkillsetDef)
	name := gjson.Get(string(jsonData), "metadata.name").String()
	path := gjson.Get(string(jsonData), "metadata.path").String()
	return fmt.Sprintf("%s/%s", path, name)
}

func MCPSkillsetAgent() string {
	return "supabase_mcp"
}
