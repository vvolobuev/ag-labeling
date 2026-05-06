#!/bin/sh
set -e
cat >/etc/ssh/sshd_config.d/99-alpha-guard-hardening.conf <<'EOF'
PasswordAuthentication no
KbdInteractiveAuthentication no
ChallengeResponseAuthentication no
PermitRootLogin prohibit-password
PubkeyAuthentication yes
X11Forwarding no
MaxAuthTries 4
LoginGraceTime 30
EOF
sshd -t
systemctl reload ssh || systemctl reload sshd
echo "SSH: password auth disabled; key auth enabled. Verify you have another session open before closing this one."
