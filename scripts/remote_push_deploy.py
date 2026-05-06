#!/usr/bin/env python3
"""
Tar local project, upload to VPS, run scripts/deploy-on-server.sh.
Requires: pip install paramiko
Usage:
  DEPLOY_SSH_PASSWORD=... python3 scripts/remote_push_deploy.py
"""

import os
import pathlib
import sys
import tarfile
import tempfile
import time

import paramiko


def stream_exec(client: paramiko.SSHClient, cmd: str, deadline_sec: float) -> int:
    """Run remote command with get_pty, stream output, tolerate slow docker builds."""
    t = client.get_transport()
    if t is None:
        raise RuntimeError("no ssh transport")

    chan = t.open_session()
    chan.settimeout(2.0)
    chan.get_pty()
    chan.exec_command(cmd)

    deadline = time.time() + deadline_sec
    while time.time() < deadline:
        if chan.recv_ready():
            try:
                chunk = chan.recv(4096)
                if chunk:
                    text = chunk.decode("utf-8", "replace")
                    sys.stdout.write(text)
                    sys.stdout.flush()
            except Exception:
                pass
        if chan.recv_stderr_ready():
            try:
                chunk = chan.recv_stderr(4096)
                if chunk:
                    sys.stderr.write(chunk.decode("utf-8", "replace"))
                    sys.stderr.flush()
            except Exception:
                pass
        if chan.exit_status_ready():
            break
        time.sleep(0.3)
    else:
        chan.close()
        raise TimeoutError("remote command timed out")

    while chan.recv_ready():
        try:
            sys.stdout.write(chan.recv(8192).decode("utf-8", "replace"))
        except Exception:
            break
    while chan.recv_stderr_ready():
        try:
            sys.stderr.write(chan.recv_stderr(8192).decode("utf-8", "replace"))
        except Exception:
            break

    return chan.recv_exit_status()


def main() -> int:
    password = os.environ.get("DEPLOY_SSH_PASSWORD", "").strip()
    if not password:
        print("Set DEPLOY_SSH_PASSWORD", file=sys.stderr)
        return 2

    host = os.environ.get("DEPLOY_SSH_HOST", "194.67.102.231").strip()
    user = os.environ.get("DEPLOY_SSH_USER", "root").strip()

    root = pathlib.Path(__file__).resolve().parents[1]

    with tempfile.NamedTemporaryFile(suffix=".tgz", delete=False) as tf:
        tpath = tf.name

    try:
        skip_dir = frozenset({"node_modules", ".git", "__pycache__", "dist"})
        with tarfile.open(tpath, "w:gz") as tar:
            for p in root.rglob("*"):
                if not p.is_file():
                    continue
                rel_parts = p.relative_to(root).parts
                if any(x in skip_dir for x in rel_parts):
                    continue
                arc = root.name + "/" + "/".join(rel_parts)
                tar.add(p, arcname=arc)
        print("tarball", tpath, "bytes", os.path.getsize(tpath))

        client = paramiko.SSHClient()
        client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
        client.connect(
            host,
            username=user,
            password=password,
            timeout=90,
            banner_timeout=90,
            auth_timeout=90,
            look_for_keys=False,
            allow_agent=False,
        )

        sftp = client.open_sftp()
        sftp.put(tpath, "/tmp/alpha-guard-ai-deploy.tgz")
        script_local = root / "scripts" / "deploy-on-server.sh"
        sftp.put(str(script_local), "/tmp/deploy-on-server.sh")
        sftp.close()
        print("upload ok")

        rc = stream_exec(
            client,
            "chmod +x /tmp/deploy-on-server.sh && /tmp/deploy-on-server.sh",
            deadline_sec=float(os.environ.get("DEPLOY_TIMEOUT_SEC", "3600")),
        )
        client.close()
        return rc
    finally:
        try:
            os.unlink(tpath)
        except OSError:
            pass


if __name__ == "__main__":
    raise SystemExit(main())
