#!/bin/bash
# VM 부팅 시 자동 실행되는 startup script
# Go 설치, 앱 디렉터리 생성, systemd 서비스 등록
set -e

LOG="/var/log/micro-trading-startup.log"
exec > >(tee -a "$LOG") 2>&1

echo "[$(date)] startup script 시작"

# ── Go 설치 ───────────────────────────────────────────────────────
apt-get update -y
apt-get install -y wget curl

GO_VERSION="1.22.4"
wget -q "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz" -O /tmp/go.tar.gz
rm -rf /usr/local/go
tar -C /usr/local -xzf /tmp/go.tar.gz
rm /tmp/go.tar.gz

cat > /etc/profile.d/go.sh << 'EOF'
export PATH=$PATH:/usr/local/go/bin
EOF
chmod +x /etc/profile.d/go.sh
echo "[$(date)] Go $GO_VERSION 설치 완료"

# ── 앱 디렉터리 생성 ──────────────────────────────────────────────
mkdir -p /opt/trading
chmod 777 /opt/trading
echo "[$(date)] 앱 디렉터리 생성 완료"

# ── systemd 서비스 등록 ───────────────────────────────────────────
cat > /etc/systemd/system/micro-trading.service << 'EOF'
[Unit]
Description=Micro Trading Backend
After=network.target

[Service]
Type=simple
WorkingDirectory=/opt/trading
EnvironmentFile=/opt/trading/.env
ExecStart=/opt/trading/server
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal
SyslogIdentifier=micro-trading

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable micro-trading
echo "[$(date)] systemd 서비스 등록 완료 (바이너리 배포 후 자동 시작)"

# ── 완료 마커 ─────────────────────────────────────────────────────
touch /tmp/startup-complete
echo "[$(date)] startup script 완료"
