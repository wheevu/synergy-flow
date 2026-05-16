#!/usr/bin/env bash
# ────────────────────────────────────────────────────────────
# SynergyFlow — EC2 bootstrap script
# ────────────────────────────────────────────────────────────
# Run this once on a fresh Ubuntu 22.04 / 24.04 EC2 instance.
# It installs Git, Docker, Docker Compose, and configures the
# required groups.  After it completes, follow the manual steps
# printed at the end.
# ────────────────────────────────────────────────────────────
set -euo pipefail

echo "==> Updating package index..."
sudo apt-get update -qq

echo "==> Installing Git..."
sudo apt-get install -y -qq git

echo "==> Installing Docker Engine..."
sudo apt-get install -y -qq ca-certificates curl
sudo install -m 0755 -d /etc/apt/keyrings
sudo curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc
sudo chmod a+r /etc/apt/keyrings/docker.asc
echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
sudo apt-get update -qq
sudo apt-get install -y -qq docker-ce docker-ce-cli containerd.io docker-compose-plugin

echo "==> Adding current user to the docker group..."
sudo usermod -aG docker "$USER"

echo ""
echo "===================================================="
echo "  Bootstrap complete!"
echo "===================================================="
echo ""
echo "Next manual steps:"
echo ""
echo "  1. Log out and back in (or run: newgrp docker)"
echo "     so the docker group takes effect."
echo ""
echo "  2. Clone the repository:"
echo "     git clone https://github.com/<YOUR_ORG>/synergy-flow.git ~/synergy-flow"
echo ""
echo "  3. Create ~/synergy-flow/.env from the example:"
echo "     cp ~/synergy-flow/.env.example ~/synergy-flow/.env"
echo "     vi ~/synergy-flow/.env"
echo ""
echo "  4. Set a strong JWT_SECRET:"
echo "     openssl rand -base64 32"
echo ""
echo "  5. Start the stack:"
echo "     cd ~/synergy-flow"
echo "     docker compose -f docker-compose.prod.yml up -d --build"
echo ""
echo "  6. Verify:"
echo "     docker compose -f docker-compose.prod.yml ps"
echo "     curl -s http://localhost/health"
echo "     curl -s http://localhost/ready"
echo ""
echo "  7. (Optional) Configure Nginx TLS:"
echo "     Obtain a certificate with certbot / Let's Encrypt and"
echo "     uncomment the HTTPS redirect block in infra/nginx/default.conf."
echo ""
echo "===================================================="
