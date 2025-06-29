# Tansive Docker Compose Setup

This directory contains the Docker Compose configuration for running Tansive with PostgreSQL.

## Services

### PostgreSQL Database

- **Image**: postgres:16
- **Container Name**: tansive-postgres
- **Database**: hatchcatalog
- **User**: tansive
- **Password**: abc@123
- **Port**: 5432

### Tansive Server

- **Image**: ghcr.io/tansive/tansivesrv:anandm-latest
- **Container Name**: tansive-server
- **Ports**:
  - 8678 (main server)
  - 9002 (endpoint server)

## Volumes

- `postgres_data`: Persistent PostgreSQL data storage
- `tansive_server_logs`: Tansive server audit logs

## Initialization

The PostgreSQL container automatically runs the following initialization scripts in order:

1. `00-create-user.sql` - Creates the catalog_api user and catalogrw role
2. `hatchcatalog.sql` - Creates all database tables and schemas

## Usage

### Start the services

```bash
cd scripts/docker
docker-compose up -d
```

### View logs

```bash
# View all logs
docker-compose logs -f

# View specific service logs
docker-compose logs -f postgres
docker-compose logs -f tansive-server
```

### Stop the services

```bash
docker-compose down
```

### Stop and remove volumes (WARNING: This will delete all data)

```bash
docker-compose down -v
```

## Configuration

The Tansive server configuration is located in `conf/tansivesrv.conf` and is mounted into the container at `/etc/tansive/tansivesrv.conf`.

Key configuration changes for Docker:

- Database host: `postgres` (container name)
- Audit log path: `/var/log/tansive/audit`

## Health Checks

The PostgreSQL container includes a health check that ensures the database is ready before the Tansive server starts. The Tansive server will wait for PostgreSQL to be healthy before starting.
