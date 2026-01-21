I am officially stopping development on the Pangolin Operator as of January 2026 because the new native **Blueprints** feature fully fulfills my needs for a declarative management approach. Below is the updated header and notice to be added to the top of your README.

***

# ⚠️ Project Discontinued (January 2026)

I’m officially stopping development on the Pangolin Operator. I originally started this project because I needed a declarative way to manage my resources, but recent updates from the official team have made this standalone tool unnecessary.

### Why I'm Moving On
- **Blueprints Fulfilled My Needs**: The official **Pangolin Blueprints** feature now provides a built-in, YAML-based declarative configuration system that handles everything I originally intended this operator to do. 
- **Official K8s Solution**: The `fosrl/pangolin` team is actively developing their own first-party Kubernetes operator. Since a native solution is being delivered, I don't see the need for this tool and won't be spending more time on it.

### Status
This repository is now **archived** and will no longer receive updates. For those looking for long-term solutions, I recommend using the official [Pangolin Blueprints](https://docs.pangolin.net/manage/blueprints) and following the official [Pangolin Project](https://github.com/fosrl/pangolin) for their upcoming Kubernetes tools.

***

# Pangolin Operator

A Kubernetes operator that manages **Pangolin tunneled reverse proxy** resources through Custom Resource Definitions (CRDs). This operator provides seamless integration between Kubernetes workloads and the Pangolin service, enabling secure tunneled access to your applications with identity-aware proxy capabilities.

[![License: AGPL v3](https://img.shields.io/badge/License-AGPL_v3-blue.svg)](https://www.gnu.org/licenses/agpl-3.0)
[![Go Report Card](https://goreportcard.com/badge/github.com/bovf/pangolin-operator)](https://goreportcard.com/report/github.com/bovf/pangolin-operator)
[![Nix](https://img.shields.io/badge/Built%20with-Nix-5277C3.svg)](https://nixos.org/)

## Overview

The Pangolin Operator simplifies the management of Pangolin organizations, tunneled sites, and proxy resources directly from Kubernetes. It automatically handles:

- **Organization Discovery**: Connect to existing Pangolin organizations or auto-discover available ones
- **Domain Management**: Automatic domain resolution and SSL certificate handling
- **Site Tunneling**: Deploy and manage Pangolin tunnel endpoints (Newt clients)
- **Resource Proxying**: Expose Kubernetes services through Pangolin's secure tunneled proxy
- **Service Binding**: Automatically create proxy resources for existing Kubernetes services

## Architecture

```
┌─────────────────────┐     ┌──────────────────────────────────────┐
│   Kubernetes CRDs   │────▶│            Kubernetes Cluster        │
│                     │     │  ┌─────────────────────────────────┐ │
│ • PangolinOrg       │     │  │       Pangolin Operator         │ │
│ • PangolinTunnel    │     │  │                                 │ │
│ • PangolinResource  │     │  │    ┌───────────┐                │ │
│ • PangolinBinding   │     │  │    │ Reconcile │                │ │
└─────────────────────┘     │  │    │   Loop    │                │ │
                            │  │    └───────────┘                │ │
                            │  └─────────────────────────────────┘ │
                            │                  │                   │
                            │                  │ REST              │
                            └──────────────────┼───────────────────┘
                                               │
                            ┌──────────────────▼───────────────────┐
                            │          Pangolin API Service        │
                            │      (api.yourpangolinisntance.com)  │
                            └──────────────────────────────────────┘
```

### Resource Hierarchy

```
PangolinOrganization
    ├── API Key Reference (Secret)
    ├── Organization ID (optional - auto-discovered)
    ├── Domain Management (auto-fetched)
    └── Default Configuration
    
PangolinTunnel
    ├── Organization Reference
    ├── Site Name + Type
    ├── Site ID (optional - for binding to existing sites)
    └── Newt Client Configuration
    
PangolinResource
    ├── Tunnel Reference
    ├── Resource Name + Protocol (HTTP/TCP/UDP)
    ├── Resource ID (optional - for binding to existing resources)
    ├── Domain Resolution (subdomain + domain selection)
    └── Target Configuration (IP + Port)
    
PangolinBinding (Service Binding)
    ├── Kubernetes Service Reference
    ├── Organization/Tunnel Reference
    └── Auto-generated PangolinResource
```

## Features

### 🏢 **Organization Management**
- **Auto-Discovery**: Automatically discover available Pangolin organizations
- **Explicit Binding**: Bind to specific organizations by ID
- **Domain Resolution**: Fetch and manage organization domains
- **API Key Management**: Secure API key storage via Kubernetes Secrets

### 🌐 **Tunnel Management**
- **Site Creation**: Create new Pangolin sites with Newt clients
- **Site Binding**: Bind to existing Pangolin sites
- **Multiple Protocols**: Support for HTTP, TCP, and UDP tunneling
- **Health Monitoring**: Track tunnel status and connectivity

### 🎯 **Resource Proxying**
- **HTTP Resources**: Create subdomain-based proxy endpoints
- **TCP/UDP Resources**: Direct port forwarding and proxy
- **Domain Flexibility**: Use domain names or IDs, with fallback to organization defaults
- **SSL Termination**: Automatic HTTPS endpoints with valid certificates

### 🔗 **Service Binding** 
- **Automatic Discovery**: Bind existing Kubernetes services to Pangolin tunnels
- **Dynamic Configuration**: Auto-generate proxy resources for services
- **Label Selectors**: Target services by labels and annotations

## Quick Start

### Prerequisites

- Kubernetes cluster (v1.20+)
- `kubectl` configured for your cluster
- Access to a Pangolin organization and API key

### Installation

1. **Install CRDs:**
```bash
kubectl apply -f https://raw.githubusercontent.com/bovf/pangolin-operator/main/config/crd/bases/tunnel.pangolin.io_pangolinorganizations.yaml
kubectl apply -f https://raw.githubusercontent.com/bovf/pangolin-operator/main/config/crd/bases/tunnel.pangolin.io_pangolintunnels.yaml
kubectl apply -f https://raw.githubusercontent.com/bovf/pangolin-operator/main/config/crd/bases/tunnel.pangolin.io_pangolinresources.yaml
kubectl apply -f https://raw.githubusercontent.com/bovf/pangolin-operator/main/config/crd/bases/tunnel.pangolin.io_pangolinbindings.yaml
```

2. **Deploy the operator:**

For production (once available):
```bash
kubectl apply -f https://raw.githubusercontent.com/bovf/pangolin-operator/main/dist/install.yaml
```

For development:
```bash
git clone https://github.com/bovf/pangolin-operator.git
cd pangolin-operator
make deploy IMG=your-registry/pangolin-operator:tag
```

3. **Create API key secret:**
```bash
kubectl create secret generic pangolin-credentials \
  --from-literal=apiKey=your-pangolin-api-key \
  -n default
```

### Basic Usage

#### 1. Create a PangolinOrganization
```yaml
apiVersion: tunnel.pangolin.io/v1alpha1
kind: PangolinOrganization
metadata:
  name: my-org
  namespace: default
spec:
  organizationId: "your-org-id"
  apiEndpoint: "https://api.pangolin.yourdomain.com"
  apiKeyRef:
    name: pangolin-credentials
    key: apiKey
  defaults:
    defaultDomain: "yourdomain.com"
    siteType: "newt"
```

#### 2. Bind a Tunnel
```yaml
apiVersion: tunnel.pangolin.io/v1alpha1
kind: PangolinTunnel
metadata:
  name: my-tunnel
  namespace: default
spec:
  organizationRef:
    name: my-org
  siteName: "k8s-cluster"
  siteType: "newt"
```

#### 3. Expose an HTTP Service
```yaml
apiVersion: tunnel.pangolin.io/v1alpha1
kind: PangolinResource
metadata:
  name: my-app
  namespace: default
spec:
  tunnelRef:
    name: my-tunnel
  name: "my-app"
  protocol: "http"
  httpConfig:
    subdomain: "app"
    # Automatically uses organization's default domain
  target:
    ip: "my-service.default.svc.cluster.local"
    port: 80
    method: "http"
```

**Access your app:** `https://app.yourdomain.com`

#### 4. Create TCP Resource (Database Proxy)
```yaml
apiVersion: tunnel.pangolin.io/v1alpha1
kind: PangolinResource
metadata:
  name: my-database
spec:
  tunnelRef:
    name: my-tunnel
  name: "my-postgres"
  protocol: "tcp"
  proxyConfig:
    proxyPort: 5432
    enableProxy: true
  target:
    ip: "postgres.default.svc.cluster.local"
    port: 5432
    method: "tcp"
```

#### 5. Service Binding (Auto-Expose Services)
```yaml
apiVersion: tunnel.pangolin.io/v1alpha1
kind: PangolinBinding
metadata:
  name: auto-expose-nginx
spec:
  serviceRef:
    name: nginx-service
    namespace: default
  tunnelRef:
    name: my-tunnel
  resourceTemplate:
    protocol: "http"
    httpConfig:
      subdomain: "nginx"
```

## Development

### 🚀 **Nix Development Environment (Recommended)**

This project includes a complete Nix flake for reproducible development environments.

#### Prerequisites

- **Nix** ([Install Nix](https://nixos.org/download.html))
- **direnv** (Optional but recommended - [Install direnv](https://direnv.net/docs/installation.html))

#### Quick Setup

1. **Clone and enter the project:**
```bash
git clone https://github.com/bovf/pangolin-operator.git
cd pangolin-operator
```

2. **Option A: Using direnv (Recommended)**
```bash
# Enable direnv for automatic environment loading
direnv allow

# Everything is automatically configured!
# You now have Go, kubectl, kind, operator-sdk, and all dev tools available
```

3. **Option B: Manual Nix Shell**
```bash
# Enter the development environment
nix develop

# All development tools are now available
go version
kubectl version --client
operator-sdk version
```

#### Available Development Commands

The Nix flake provides convenient development commands:

**🔧 Setup Local Kubernetes Cluster:**
```bash
# Creates a local kind cluster with ingress support
nix run .#setup-cluster

# Exports kubeconfig to ./.kubeconfig
export KUBECONFIG=$PWD/.kubeconfig
```

**🚀 Build and Deploy Operator:**
```bash
# Builds, containerizes, and deploys the operator to local cluster
nix run .#deploy-operator

# Check deployment status
kubectl -n pangolin-operator-system get pods
kubectl -n pangolin-operator-system logs deployment/pangolin-operator-controller-manager -c manager
```

**🧹 Cleanup Everything:**
```bash
# Removes cluster, images, and temporary files
nix run .#cleanup
```

#### Development Tools Included

The Nix environment provides:

- **Go Ecosystem**: `go`, `gopls`, `go-tools`, `gofumpt`, `golangci-lint`
- **Kubernetes**: `kubectl`, `helm`, `kind`, `kustomize`, `operator-sdk`, `kubebuilder`
- **Container Tools**: `docker` (with Colima support on macOS ARM)
- **Utilities**: `jq`, `yq-go`, `curl`, `tree`, `bat`, `fd`, `ripgrep`

#### Platform-Specific Notes

**🍎 macOS (Apple Silicon):**
- Automatically configures Colima for Docker compatibility
- Uses optimized operator-sdk plugins (`go/v4`)
- Handles Apple Silicon-specific considerations

**🐧 Linux:**
- Uses native Docker when available
- Optimized for common Linux distributions

#### Local Development Workflow

1. **Start Development Environment:**
```bash
cd pangolin-operator
direnv allow  # or nix develop
```

2. **Setup Local Cluster:**
```bash
nix run .#setup-cluster
export KUBECONFIG=$PWD/.kubeconfig
```

3. **Make Changes and Test:**
```bash
# Edit code, then deploy
nix run .#deploy-operator

# Test your changes
kubectl apply -f config/samples/
kubectl get pangolinorganization,pangolintunnel,pangolinresource
```

4. **Iterate:**
```bash
# Make more changes, rebuild/redeploy
nix run .#deploy-operator

# Check logs
kubectl -n pangolin-operator-system logs deployment/pangolin-operator-controller-manager -c manager -f
```

5. **Cleanup When Done:**
```bash
nix run .#cleanup
```

### 🔧 **Environment Configuration**

#### direnv Setup (.envrc)

Create `.envrc` file for automatic environment loading:
```bash
use flake
```

### 🐳 **Docker Development** 

**macOS with Apple Silicon:**
```bash
# Colima is automatically managed by the Nix environment
# Manual control if needed:
colima start --cpu 4 --memory 6 --arch aarch64
export DOCKER_HOST="unix://$HOME/.colima/default/docker.sock"
```

**Linux:**
```bash
# Standard Docker
docker --version
# OR Podman compatibility
export DOCKER_CMD=podman
```

### 📋 **Traditional Development (Without Nix)**

If you prefer not to use Nix:

#### Prerequisites
- Go 1.22+
- Docker
- kubectl
- operator-sdk
- kind (for local testing)

#### Setup
```bash
git clone https://github.com/bovf/pangolin-operator.git
cd pangolin-operator

# Install CRDs
make install

# Run locally
make run

# Generate code after API changes
make generate
make manifests

# Build and test
make build
make test

# Deploy to cluster
make docker-build docker-push IMG=your-registry/pangolin-operator:tag
make deploy IMG=your-registry/pangolin-operator:tag
```

## Advanced Configuration

### Domain Resolution Options

The operator supports flexible domain resolution:

```yaml
httpConfig:
  subdomain: "api"
  # Option 1: Use specific domain ID
  domainId: "domain1"
  
  # Option 2: Use domain name (auto-resolved to ID)
  domainName: "yourdomain.com"
  
  # Option 3: Use organization default (no domain specified)
  # Falls back to organization's defaultDomain
```

### Binding to Existing Resources

Bind to existing Pangolin organizations, sites, or resources:

```yaml
# Bind to existing organization
spec:
  organizationId: "existing-org-123"

# Bind to existing site  
spec:
  siteId: 456

# Bind to existing resource
spec:
  resourceId: "existing-resource-789"
```

### Service Discovery and Binding

Automatically expose Kubernetes services:

```yaml
apiVersion: tunnel.pangolin.io/v1alpha1
kind: PangolinBinding
metadata:
  name: auto-expose-all
spec:
  selector:
    matchLabels:
      expose: "pangolin"
  tunnelRef:
    name: my-tunnel
  resourceTemplate:
    protocol: "http"
    httpConfig:
      # Subdomain will be auto-generated from service name
      subdomain: "{{ .ServiceName }}"
```

## Status and Monitoring

All resources provide comprehensive status information:

```bash
# Check organization status
kubectl get pangolinorganization my-org -o wide

# Check tunnel status
kubectl get pangolintunnel my-tunnel -o wide

# Check resource status with URLs
kubectl get pangolinresource my-web-app -o wide
NAME         RESOURCE ID   PROTOCOL   SUBDOMAIN   FULL DOMAIN              URL                          STATUS   BINDING MODE   AGE
my-web-app   32           http       app         app.yourdomain.com      https://app.yourdomain.com    Ready    Created        5m

# Check detailed status
kubectl describe pangolinresource my-web-app
```

## Troubleshooting

### Common Issues

**Resource stays in "Waiting" status:**
```bash
# Check if organization and tunnel are ready
kubectl get pangolinorganization,pangolintunnel

# Check operator logs
kubectl logs -l app=pangolin-operator -n pangolin-operator-system
```

**Domain resolution fails:**
```bash
# Check organization domains
kubectl get pangolinorganization my-org -o jsonpath='{.status.domains}'

# Verify domain name/ID in resource spec
kubectl describe pangolinresource my-resource
```

**Target creation fails:**
```bash
# Verify service/endpoint exists
kubectl get service my-service
kubectl get endpoints my-service

# Check resource logs
kubectl describe pangolinresource my-resource
```

**Nix/direnv Issues:**
```bash
# Reload direnv environment
direnv reload

# Clear Nix caches if needed
nix-collect-garbage
nix develop --refresh

# Colima issues on macOS
colima stop && colima start --cpu 4 --memory 6 --arch aarch64
```

### Debug Mode

Enable debug logging:
```bash
kubectl patch deployment pangolin-operator-controller-manager \
  -n pangolin-operator-system \
  --type='json' \
  -p='[{"op": "replace", "path": "/spec/template/spec/containers/0/args", "value": ["--zap-log-level=debug"]}]'
```

## Examples

Complete examples are available in the [`config/samples/`](config/samples/) directory:

- [Basic Organization Setup](config/samples/tunnel_v1alpha1_pangolinorganization.yaml)
- [HTTP Resource Example](config/samples/tunnel_v1alpha1_pangolinresource_http.yaml)  
- [TCP Resource Example](config/samples/tunnel_v1alpha1_pangolinresource_tcp.yaml)
- [Service Binding Example](config/samples/tunnel_v1alpha1_pangolinbinding.yaml)
- [Complete Application Stack](config/samples/complete-example.yaml)

## Contributing

### Development Workflow

1. **Fork and Clone:**
```bash
git clone https://github.com/yourusername/pangolin-operator.git
cd pangolin-operator
```

2. **Setup Development Environment:**
```bash
direnv allow  # or nix develop
nix run .#setup-cluster
```

3. **Create Feature Branch:**
```bash
git checkout -b feature/amazing-feature
```

4. **Make Changes and Test:**
```bash
# Edit code
vim internal/controller/...

# Test locally
nix run .#deploy-operator
kubectl apply -f config/samples/
```

5. **Submit Pull Request:**
```bash
git add .
git commit -m 'Add amazing feature'
git push origin feature/amazing-feature
```

### Code Standards

- **Go**: Follow standard Go conventions and use `gofumpt` for formatting
- **YAML**: Use consistent indentation (2 spaces)
- **Documentation**: Update README and API docs for new features
- **Tests**: Add unit tests for new functionality

## Security Considerations

- **API Keys**: Store API keys in Kubernetes Secrets, never in plain text
- **RBAC**: The operator requires appropriate cluster permissions
- **Network Policies**: Consider network policies for tunnel endpoints
- **Resource Limits**: Set appropriate resource limits for tunnel deployments
- **Supply Chain**: Nix provides reproducible builds and dependency pinning

## API Reference

Detailed API reference documentation is available at [docs/api.md](docs/api.md).

## License

Copyright (C) 2025 github.com/bovf

This program is free software: it can be redistributed and/or modified under the terms of the GNU Affero General Public License as published by the Free Software Foundation, either version 3 of the License, or (at the option) any later version.

This program is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU Affero General Public License for more details.

A copy of the GNU Affero General Public License should be included with this program. If not, see https://www.gnu.org/licenses/.

Third‑party code bundled in this repository may be licensed under different terms (for example, Apache‑2.0 for Kubernetes libraries). Such components retain their original licenses; see the corresponding LICENSE/NOTICE files in their source directories.

## Community

- **Issues**: [GitHub Issues](https://github.com/bovf/pangolin-operator/issues)
- **Discussions**: [GitHub Discussions](https://github.com/bovf/pangolin-operator/discussions)
- **Pangolin Project**: [fosrl/pangolin](https://github.com/fosrl/pangolin)
- **Nix Community**: [NixOS Discourse](https://discourse.nixos.org/)

---

*Built with ❤️ using [Operator SDK](https://sdk.operatorframework.io/) and [Nix](https://nixos.org/) for reproducible development.*
