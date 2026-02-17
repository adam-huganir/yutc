# Docker Compose Multi-Environment Example

This example shows how to generate different `docker-compose.yml` files for dev, staging, and production from **simple configuration files** - no Docker Compose knowledge required!

## Why This Example?

Instead of maintaining separate Docker Compose files for each environment (which leads to duplication and drift), you maintain:
- **Simple config files** with just the values that change (ports, passwords, resource limits)
- **One template** that handles all the Docker Compose complexity

## What You Get

A complete microservices stack:
- **Web server** (nginx) - serves your frontend
- **API server** (Node.js) - your backend application
- **Database** (PostgreSQL) - data storage
- **Cache** (Redis) - session/data caching
- **Dev tools** (Adminer) - database UI for development only

Each environment automatically gets the right configuration:
- **Dev**: Debug logging, hot-reload, database UI, relaxed resources
- **Staging**: Production-like setup for testing
- **Production**: Multiple replicas, strict resource limits, minimal logging

## Directory Structure

```
docker-compose-multi-env/
├── data/
│   ├── base.yaml       # Shared settings (image versions, defaults)
│   ├── dev.yaml        # Dev-specific values (ports, debug flags)
│   ├── staging.yaml    # Staging values
│   └── prod.yaml       # Production values (replicas, limits)
├── docker-compose.yaml.tmpl  # The template (you rarely need to touch this)
└── README.md
```

## The Config Files Are Simple!

Here's what a config file looks like - just plain key-value pairs:

```yaml
# dev.yaml - easy to read and modify!
environment: development

# Ports
web_port: 8080
api_port: 3000
db_port: 5432

# Database
db_name: myapp_dev
db_user: devuser
db_password: devpass123

# Features
log_level: debug
enable_debug: true
enable_dev_tools: true  # Adds database UI
```

No Docker Compose syntax, no complex nesting - just simple configuration!

## Quick Start

### 1. Generate Your Environment

Pick your environment and run one command:

**Development:**
```bash
yutc -d examples/docker-compose-multi-env/data/base.yaml \
     -d examples/docker-compose-multi-env/data/dev.yaml \
     -o docker-compose.yml \
     examples/docker-compose-multi-env/docker-compose.yaml.tmpl
```

**Staging:**
```bash
yutc -d examples/docker-compose-multi-env/data/base.yaml \
     -d examples/docker-compose-multi-env/data/staging.yaml \
     -o docker-compose.yml \
     examples/docker-compose-multi-env/docker-compose.yaml.tmpl
```

**Production:**
```bash
yutc -d examples/docker-compose-multi-env/data/base.yaml \
     -d examples/docker-compose-multi-env/data/prod.yaml \
     -o docker-compose.yml \
     examples/docker-compose-multi-env/docker-compose.yaml.tmpl
```

### 2. Start Your Services

```bash
docker-compose up
```

That's it! Your environment-specific stack is running.

## Common Tasks

### Change a Setting

Just edit the config file! For example, to change the dev database password:

```yaml
# data/dev.yaml
db_password: my_new_password  # Changed this line
```

Then regenerate:
```bash
yutc -d examples/docker-compose-multi-env/data/base.yaml \
     -d examples/docker-compose-multi-env/data/dev.yaml \
     -o docker-compose.yml \
     examples/docker-compose-multi-env/docker-compose.yaml.tmpl
```

### Override a Value Without Editing Files

Use `--set` for one-off changes:

```bash
# Use a different port temporarily
yutc -d examples/docker-compose-multi-env/data/base.yaml \
     -d examples/docker-compose-multi-env/data/dev.yaml \
     --set '.api_port=4000' \
     -o docker-compose.yml \
     examples/docker-compose-multi-env/docker-compose.yaml.tmpl

# Inject a secure password from environment
yutc -d examples/docker-compose-multi-env/data/base.yaml \
     -d examples/docker-compose-multi-env/data/prod.yaml \
     --set ".db_password=$DB_PASSWORD" \
     -o docker-compose.yml \
     examples/docker-compose-multi-env/docker-compose.yaml.tmpl
```

### Disable a Service

Set the feature flag to `false`:

```bash
# Run without Redis cache
yutc -d examples/docker-compose-multi-env/data/base.yaml \
     -d examples/docker-compose-multi-env/data/dev.yaml \
     --set '.enable_cache=false' \
     -o docker-compose.yml \
     examples/docker-compose-multi-env/docker-compose.yaml.tmpl
```

## How It Works

### 1. Config Files Merge

`base.yaml` + `dev.yaml` = your dev configuration

Values in later files override earlier ones:
```yaml
# base.yaml
api_cpu_limit: "1.0"     # Default

# dev.yaml
api_cpu_limit: "0.5"     # Override for dev (less resources needed)
```

### 2. Feature Flags Control Services

Simple boolean flags enable/disable entire services:
```yaml
# dev.yaml
enable_dev_tools: true   # Adds Adminer database UI

# prod.yaml
enable_dev_tools: false  # No dev tools in production
```

### 3. Template Handles Complexity

The template converts your simple config into proper Docker Compose:
```yaml
# Your config:
web_port: 8080

# Becomes in docker-compose.yml:
services:
  web:
    ports:
      - "8080:80"
```

You don't need to know Docker Compose syntax - just set the values!

### 4. Environment-Aware Behavior

The template automatically adjusts based on your settings:
- `enable_hot_reload: true` → uses `npm run dev`
- `enable_hot_reload: false` → uses `npm start`
- `api_replicas: 3` → creates 3 API instances
- No `api_replicas` → creates 1 instance

## Benefits

✅ **No Docker Compose expertise needed** - just edit simple config files
✅ **No duplication** - one template, multiple environments
✅ **Easy to review** - config files are small and readable
✅ **Version controlled** - track all environment configs in git
✅ **Consistent** - all environments use the same structure
✅ **Flexible** - override anything with `--set` for CI/CD or testing

## Customizing for Your Project

### Add Your Own Settings

Just add new keys to your config files:
```yaml
# data/dev.yaml
my_custom_setting: some_value
api_timeout: 30
```

Then use them in the template:
```yaml
# docker-compose.yaml.tmpl
environment:
  API_TIMEOUT: {{ .api_timeout }}
```

### Add More Environments

Create a new config file:
```yaml
# data/qa.yaml
environment: qa
web_port: 9090
db_name: myapp_qa
# ... other QA-specific values
```

### Change the Stack

Edit `base.yaml` to enable/disable services:
```yaml
enable_cache: false      # Don't need Redis
enable_database: true    # Keep PostgreSQL
```

## Security Best Practices

⚠️ **Important**: This example uses plaintext passwords for demonstration.

For real deployments:
```bash
# Inject secrets from environment variables
export DB_PASSWORD=$(vault read -field=password secret/db)
yutc -d data/base.yaml -d data/prod.yaml \
     --set ".db_password=$DB_PASSWORD" \
     -o docker-compose.yml \
     docker-compose.yaml.tmpl

# Or from a secrets file not in git
yutc -d data/base.yaml -d data/prod.yaml \
     -d /secure/secrets.yaml \
     -o docker-compose.yml \
     docker-compose.yaml.tmpl
```

## Next Steps

1. **Try it out**: Generate a dev environment and run `docker-compose up`
2. **Modify a setting**: Change a port in `data/dev.yaml` and regenerate
3. **Add a new environment**: Copy `data/dev.yaml` to `data/local.yaml` and customize
4. **Integrate with CI/CD**: Use `--set` to inject secrets and generate configs in your pipeline

## Learn More

This example demonstrates:
- **Data merging** (`-d` flag with multiple files)
- **Runtime overrides** (`--set` flag)
- **Conditional templating** (`if` statements in templates)
- **Feature flags** (boolean values to enable/disable features)

Check out the main [yutc README](../../README.md) for more advanced features!
