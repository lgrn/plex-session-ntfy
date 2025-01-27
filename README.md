# plex-session-ntfy

The following values can be set in `.env`, or however you prefer to set
environment variables:

- `PSN_PLEX_IP` (optional, default:"127.0.0.1")
- `PSN_PLEX_TOKEN` (required)
- `PSN_NTFY_TOPIC_URL` (required)
- `PSN_IGNORED_USER` (optional)
- `PSN_CHECK_INTERVAL` (required, default:"30s")

## systemd service

```ini
[Unit]
Description=Plex Session Notifier Service
After=network.target
Requires=plexmediaserver.service

[Service]
Type=simple
EnvironmentFile=/root/plex-session-ntfy/.env
ExecStart=/usr/local/go/bin/go run /root/plex-session-ntfy/psn.go
Restart=on-failure
User=root
WorkingDirectory=/root/plex-session-ntfy/

[Install]
WantedBy=multi-user.target
```

```bash
systemctl daemon-reload
systemctl enable plex-session-ntfy --now
```

Yes, you can obviously compile the binary instead of running
`go run`.
