# How to setup a capability based RBAC system for MCP servers using Tansive.

In this guide, we'll walk through the steps of how to create an role-based access system on top of MCP servers for public services such as Github using Tansive. Tansive provides a policy-based managed MCP endpoint with audit logs for use with tools such as Cursor, Claude, Windsurf, etc. You can, of course, use the endpoint with your own agent applications.

Tansive allows you to assign _Capability_ tags for Skills. Views can then be created based on Allow/Deny rules expressed in terms of capabilities.

Below are two examples of how to use Tansive to create a fine-grained RBAC system on top of MCP tools - Supabase MCP Server and Github MCP Server.

When creating policy based tool filters, it is not necessary to always think in terms of roles. It could also be use-case driven, workflow driven, or environment (dev, stage, etc) specific policies.

## Table of Contents

- [Supabase](#supabase)
  - [Capabilities and Tools](#capabilities-and-tools)
  - [Roles and Permissions](#roles-and-permissions)
    - [Support Agent](#support-agent)
    - [App Developer](#app-developer)
    - [DevOps Engineer](#devops-engineer)
- [GitHub](#github)
  - [Capabilities and Tools](#capabilities-and-tools-1)
  - [Roles and Permissions](#roles-and-permissions-1)
    - [Developer](#developer)
    - [DevOps Engineer](#devops-engineer-1)
    - [Tester](#tester)
    - [Engineering Manager](#engineering-manager)
    - [Security Engineer](#security-engineer)
    - [Read-Only](#read-only)
    - [Full Access](#full-access)

## Supabase

We import tools provided by the Supabase MCP Server as Tansive Skills and assign tags.
We then create `app-developer`, `support-agent`, and `devops-engineer` roles.
skillset-supabase.yaml and skillset-supabase-views.yaml implement the definitions.

### Capabilities and Tools

#### Database Operations (`supabase.sql.*`)

- **`supabase.sql.query`** - Execute SQL queries with validation
  - `execute_sql` - Execute validated SQL queries against the database
- **`supabase.sql.develop`** - Development-level database operations
  - Limited access to sensitive tables (integration_tokens)
  - Cannot perform destructive operations (DROP)
- **`supabase.sql.devopsadmin`** - Full database administration
  - Complete access to all tables including sensitive data
  - Can perform all database operations

#### Deployment Management (`supabase.deploy.*`)

- **`supabase.deploy.read`** - Read deployment information
  - `get_advisors` - Get security and performance advisors
  - `get_logs` - Get recent project logs
  - `list_branches` - List all database branches
  - `list_edge_functions` - List all Edge Functions
  - `list_extensions` - List all database extensions
  - `list_migrations` - List all database migrations
- **`supabase.deploy.write`** - Write deployment operations
  - `apply_migration` - Apply a database migration
  - `create_branch` - Create a new database branch
  - `delete_branch` - Delete a database branch
  - `deploy_edge_function` - Deploy an Edge Function
  - `merge_branch` - Merge a branch into production
  - `rebase_branch` - Rebase a branch onto production
  - `reset_branch` - Reset a branch to a migration version

#### Project Management (`supabase.project.*`)

- **`supabase.project.read`** - Read project information
  - `get_anon_key` - Get the anonymous API key
  - `get_project_url` - Get the project API URL

#### Code Generation (`supabase.codegen.*`)

- **`supabase.codegen.read`** - Generate code artifacts
  - `generate_typescript_types` - Generate TypeScript types

#### Documentation (`supabase.docs.*`)

- **`supabase.docs.read`** - Access documentation
  - `search_docs` - Search the Supabase documentation

#### MCP Server (`supabase.mcp.*`)

- **`supabase.mcp.use`** - Use Supabase MCP server tools
  - `supabase_mcp` - Direct access to Supabase MCP server
  - `list_tables` - List database tables
  - `validate_sql` - Validate SQL input (custom validator)

### Roles and Permissions

#### Support Agent

- **Capabilities**: `supabase.sql.query`, `supabase.mcp.use`
- **Purpose**: Basic database querying for support operations
- **Access**: Limited to support-related tables (support_tickets, support_messages)

#### App Developer

- **Capabilities**: `supabase.sql.query`, `supabase.sql.develop`, `supabase.deploy.read`, `supabase.codegen.read`, `supabase.docs.read`, `supabase.project.read`, `supabase.mcp.use`
- **Purpose**: Full development workflow with safety restrictions
- **Access**: Can read deployment info, generate types, access docs, but cannot perform destructive operations

#### DevOps Engineer

- **Capabilities**: All capabilities including `supabase.deploy.write`, `supabase.sql.devopsadmin`
- **Purpose**: Complete administrative access for deployment and database management
- **Access**: Full access to all tables and operations, including sensitive data

---

## GitHub

Github provides a broader set of tools. We define 8 different capabilities.  
Then we create 7 roles and assign them capabilities relevant to the role.

### Capabilities and Tools

#### Basic Read Operations (`github.read.basic`)

- **User Profile**: `get_me` - Get user profile information
- **Issues**: `get_issue`, `list_issues` - View and list issues
- **Pull Requests**: `get_pull_request`, `list_pull_requests` - View and list pull requests
- **Repository Content**: `get_file_contents`, `list_branches`, `list_commits` - Access repository content
- **Search**: `search_code`, `search_repositories` - Search code and repositories

#### Code Development (`github.write.code`)

- **Pull Requests**: `create_pull_request`, `update_pull_request` - Manage pull requests
- **Branches**: `create_branch` - Create new branches
- **Files**: `create_or_update_file`, `delete_file`, `push_files` - Manage repository files
- **Issues**: `create_issue`, `update_issue`, `add_issue_comment` - Manage issues and comments

#### Pull Request Management (`github.develop.code`)

- **Reviews**: `create_pending_pull_request_review`, `submit_pending_pull_request_review`, `create_and_submit_pull_request_review` - Manage PR reviews
- **Comments**: `add_comment_to_pending_review`, `get_pull_request_comments` - Manage review comments
- **PR Information**: `get_pull_request_reviews`, `get_pull_request_diff`, `get_pull_request_files`, `get_pull_request_status` - Access PR details
- **PR Operations**: `merge_pull_request`, `update_pull_request_branch` - Perform PR operations
- **Copilot Integration**: `request_copilot_review`, `assign_copilot_to_issue` - GitHub Copilot features

#### Workflow Management (`github.workflow.manage`)

- **Workflows**: `list_workflows`, `list_workflow_runs`, `get_workflow_run` - Manage workflows
- **Execution**: `run_workflow`, `rerun_workflow_run`, `rerun_failed_jobs`, `cancel_workflow_run` - Control workflow execution
- **Jobs**: `list_workflow_jobs`, `get_job_logs` - Manage workflow jobs
- **Logs**: `get_workflow_run_logs`, `delete_workflow_run_logs` - Access workflow logs
- **Artifacts**: `list_workflow_run_artifacts`, `download_workflow_run_artifact` - Manage workflow artifacts
- **Usage**: `get_workflow_run_usage` - Monitor workflow usage

#### Security Management (`github.security.manage`)

- **Code Scanning**: `list_code_scanning_alerts`, `get_code_scanning_alert` - Manage code scanning alerts
- **Dependabot**: `list_dependabot_alerts`, `get_dependabot_alert` - Manage Dependabot alerts
- **Secret Scanning**: `list_secret_scanning_alerts`, `get_secret_scanning_alert` - Manage secret scanning alerts

#### Content Management (`github.content.manage`)

- **Commits**: `get_commit` - Access commit information
- **Tags**: `get_tag`, `list_tags` - Manage repository tags
- **Repository**: `fork_repository` - Fork repositories
- **Search**: `search_issues`, `search_pull_requests` - Advanced search capabilities
- **Comments**: `get_issue_comments` - Access issue comments
- **Discussions**: `list_discussions`, `get_discussion`, `get_discussion_comments`, `list_discussion_categories` - Manage discussions

#### Administrative Operations (`github.admin.manage`)

- **Repositories**: `create_repository` - Create new repositories
- **Search**: `search_orgs`, `search_users` - Search organizations and users

#### Notification Management (`github.test.manage`)

- **Notifications**: `list_notifications`, `get_notification_details`, `dismiss_notification` - Manage notifications
- **Subscriptions**: `manage_notification_subscription`, `manage_repository_notification_subscription` - Manage notification subscriptions
- **Bulk Operations**: `mark_all_notifications_read` - Bulk notification management

### Roles and Permissions

#### Developer

- **Capabilities**: `github.read.basic`, `github.write.code`, `github.develop.code`
- **Purpose**: Full development workflow including code writing and PR management
- **Access**: Can read repositories, write code, create PRs, and manage reviews

#### DevOps Engineer

- **Capabilities**: `github.read.basic`, `github.workflow.manage`, `github.security.manage`, `github.admin.manage`
- **Purpose**: Infrastructure and security management
- **Access**: Can manage workflows, security alerts, and administrative tasks

#### Tester

- **Capabilities**: `github.read.basic`, `github.write.code`, `github.test.manage`
- **Purpose**: Testing and quality assurance
- **Access**: Can read repositories, create issues, and manage notifications

#### Engineering Manager

- **Capabilities**: `github.read.basic`, `github.develop.code`, `github.content.manage`, `github.admin.manage`
- **Purpose**: Team leadership and project management
- **Access**: Can manage PRs, content, and administrative tasks

#### Security Engineer

- **Capabilities**: `github.read.basic`, `github.security.manage`, `github.workflow.manage`
- **Purpose**: Security-focused operations
- **Access**: Can manage security alerts and workflows

#### Read-Only

- **Capabilities**: `github.read.basic`
- **Purpose**: View-only access for stakeholders
- **Access**: Can only read repositories and content

#### Full Access

- **Capabilities**: All GitHub capabilities
- **Purpose**: Complete administrative access
- **Access**: Can perform all GitHub operations

---

## Usage

Assuming you have already run catalog_setup/setup.sh, you should have a `dev` variant.

For Github, pick the latest release from the [Github MCP Server Repository](https://github.com/github/github-mcp-server/releases)
Go to "Assets" and download the package specific to your OS using `curl`. Don't download via browser.

For example:

`curl -LO https://github.com/github/github-mcp-server/releases/download/v0.8.0/github-mcp-server_Linux_i386.tar.gz`

Extract the package: `tar -xvf github-mcp-server_Linux_i386.tag.gz` for example.

Add the path to the github-mcp-server in your .env file as described in the Getting Started section of the README in the root of this repository.

Then install the SkillSet and Views as below.

```bash
tansive create -f catalog_setup/skillset-github.yaml -v dev
tansive create -f catalog_setup/skillset-github-views.yaml -v dev

# create a session using a 'developer' view.
tansive session create /demo-skillsets/github-demo/github-mcp-server --view developer

# tansive will return an MCP endpoint and Access token

```
