# Deployment Setup Guide

This guide explains how to configure GitHub secrets for the existing GitHub Actions automated deployment.

## Required GitHub Secrets

Navigate to GitHub repository → Settings → Secrets and variables → Actions, then add these secrets:

### Docker Hub Credentials
- `DOCKER_USERNAME` - Docker Hub username
- `DOCKER_PASSWORD` - Docker Hub password or access token

### Production Environment
- `PROD_HOST` - Production server IP address or domain
- `PROD_USERNAME` - SSH username for production server
- `PROD_SSH_KEY` - Private SSH key for production server access
- `PROD_SSH_PORT` - SSH port (optional, defaults to 22)
- `PROD_PORT` - Port to expose the application (optional, defaults to 8080)
- `PROD_ENV_FILE` - Path to the environment file on server (optional, defaults to `/opt/app/.env`)

## Server Setup

### 1. Install Docker

```bash
# Update system
sudo apt update && sudo apt upgrade -y

# Install Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh

# Add user to docker group
sudo usermod -aG docker $USER

# Start and enable Docker
sudo systemctl start docker
sudo systemctl enable docker
```

### 2. Create Environment Files

Create the environment file on the server, `.env.example` is provided for reference:

**Production (`/opt/app/.env` or value set to `PROD_ENV_FILE`):**
```bash
sudo mkdir -p /opt/app
sudo nano /opt/app/.env
```

### 3. SSH Key Setup

Generate SSH key pairs for GitHub Actions:

```bash
# Generate SSH key pair
ssh-keygen -t ed25519 -C "github-actions" -f ~/.ssh/github-actions

# Add public key to authorized_keys
cat ~/.ssh/github-actions.pub >> ~/.ssh/authorized_keys

# Copy private key content for GitHub secret
cat ~/.ssh/github-actions
```

Copy the private key content and add it to GitHub secrets `PROD_SSH_KEY`.

## Workflow Triggers

- **Production**: Automatically deploys when code is pushed/merged to `main` branch
- **Manual**: Workflow can be triggered manually from GitHub Actions tab

## Docker Image Tags

The workflows create multiple tags for each build:
- `production` - Environment-specific tag
- `latest` - Latest tag
- `main-<commit-sha>` - Commit-specific tags

## Troubleshooting

### Common Issues

1. **SSH Connection Failed**
    - Verify server IP and SSH port
    - Check SSH key format (should be private key, not public)
    - Ensure user has sudo privileges or Docker group membership

2. **Docker Login Failed**
    - Verify Docker Hub credentials
    - Consider using access tokens instead of passwords

3. **Port Already in Use**
    - Check if containers are already running: `docker ps`
    - Stop existing containers: `docker stop <container-name>`

4. **Environment File Not Found**
    - Ensure the environment file exists on the server at the specified path
    - Check file permissions: `sudo chmod 644 /opt/app/.env`

### Monitoring Deployments

Check deployment status:
```bash
# View running containers
docker ps

# Check container logs
docker logs <container-name>

# Monitor resource usage
docker stats
```
