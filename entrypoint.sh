#!/bin/sh
chown -R appuser:appuser /data/keys
exec su-exec appuser /app/ssh.paullj.com "$@"
