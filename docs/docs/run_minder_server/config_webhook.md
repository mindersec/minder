---
title: Configuring a Webhook
sidebar_position: 70
---

# Configuring a Webhook
Minder allows a webhook to be configured on the repository provider to respond to provider events. Currently, Minder only supports GitHub. 
The webhook allows GitHub to notify Minder when certain events occur in your repositories.
To configure the webhook, Minder needs to be accessible from the internet. If you are running the server locally, you 
can use a service like [ngrok](https://ngrok.com/) to expose your local server to the internet.

Here are the steps to configure the webhook:

1. **Expose your local server:** If you are running the server locally, start ngrok or a similar service to expose your 
local server to the internet. Note down the URL provided by ngrok (it will look something like `https://<random-hash>.ngrok.io`).
Make sure to expose the port that Minder is running on (by default, this is port `8080`).

2. **Update the Minder configuration:** Open your `server-config.yaml` file and update the `webhook-config` section with 
the ngrok URL Minder is running on. The `external_webhook_url` should point to the `/api/v1/webhook/github`
endpoint on your Minder server, and the `external_ping_url` should point to the `/api/v1/health` endpoint. The `webhook_secret`
should match the secret configured in the GitHub webhook (under `github.payload_secret`).

```yaml
webhook-config:
    external_webhook_url: "https://<ngrok-url>/api/v1/webhook/github"
    external_ping_url: "https://<ngrok-url>/api/v1/health"
    webhook_secret: "your-password" # Should match the secret configured in the GitHub webhook (github.payload_secret)
```

After these steps, your Minder server should be ready to receive webhook events from GitHub, and add webhooks to repositories.

In case you need to update the webhook secret, you can do so by putting the
new secret in `webhook-config.webhook_secret` and for the duration of the
migration, the old secret(s) in a file referenced by
`webhook-config.previous_webhook_secret_file`. The old webhook secrets will
then only be used to verify incoming webhooks messages, not for creating or
updating webhooks and can be removed after the migration is complete.

In order to rotate webhook secrets, you can use the `minder-server` CLI tool to update the webhook secret.

```bash
minder-server webhook update -p github
```

Note that the command simply replaces the webhook secret on the provider
side. You will still need to update the webhook secret in the server configuration
to match the provider's secret.