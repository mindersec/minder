---
title: Customizing Email Templates
sidebar_position: 70
---

When users are invited to an organization or project, Minder sends them an invitation email. By default, these email templates contain Stacklok branding, logos, and hardcoded URLs for Terms of Service and Privacy Policy.

Minder builds these templates directly into the container under the `/var/run/ko/templates/` directory using `ko`. If you are self-hosting Minder and wish to override these templates to include your organization's own branding, policies, or layout, you can inject your custom templates via Kubernetes volume mounts.

## Available Templates

Minder includes the following email templates that can be customized:

- `invite-email.html.tmpl` - HTML version of invitation emails
- `invite-email.txt.tmpl` - Plain text version of invitation emails

## Template Variables

All email templates have access to the following variables:

| Variable | Description | Example |
|----------|-------------|---------|
| `{{.AdminName}}` | Name of the user sending the invitation | "John Doe" |
| `{{.OrganizationName}}` | Name of the organization | "Acme Corp" |
| `{{.OrganizationId}}` | UUID of the organization | "123e4567-e89b-12d3-a456-426614174000" |
| `{{.InvitationCode}}` | Unique invitation code | "abc123def456" |
| `{{.InvitationURL}}` | Full URL to accept the invitation | "https://minder.example.com/invite/abc123" |
| `{{.RecipientEmail}}` | Email address of the invitee | "user@example.com" |
| `{{.MinderURL}}` | Base URL of your Minder instance | "https://minder.example.com" |
| `{{.TermsURL}}` | Terms of Service URL | "https://example.com/terms" |
| `{{.PrivacyURL}}` | Privacy Policy URL | "https://example.com/privacy" |
| `{{.SignInURL}}` | Sign-in URL for Minder | "https://minder.example.com/login" |
| `{{.RoleName}}` | Role being assigned | "admin", "editor", "viewer" |
| `{{.RoleVerb}}` | Action the role can perform | "manage", "contribute to", "view" |

> **Note**: We recommend hardcoding parameters that don't need to vary between environments, such as your domain name. For example, instead of using `{{.InvitationURL}}` or `{{.SignInURL}}`, you construct these URLs explicitly in your template using your known domain and the `{{.InvitationCode}}` directly (e.g., `https://minder.your-domain.com/invite/{{.InvitationCode}}`). This is especially important for `{{.TermsURL}}` and `{{.PrivacyURL}}`, which currently default to Stacklok's URLs in the application code.

This guide outlines two distinct ways to override these templates when running Minder on Kubernetes.

## 1. The Standard Approach: Using ConfigMaps

For standard Kubernetes deployments, the most straightforward approach is to store your custom template inside a `ConfigMap` and mount it directly over the default file in the container.

### Create a ConfigMap

First, construct your custom template and create a `ConfigMap`. Give your file the same name as the target template you wish to override (e.g., `invite-email.html.tmpl`).

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: minder-custom-email-template
data:
  invite-email.html.tmpl: |
    <!DOCTYPE html>
    <html>
      <head>
        <title>Invitation to {{.OrganizationName}}</title>
        <style>
          body { font-family: Arial, sans-serif; margin: 40px; }
          .header { background-color: #your-brand-color; padding: 20px; text-align: center; }
          .content { padding: 20px; }
          .button { background-color: #your-accent-color; color: white; padding: 10px 20px; text-decoration: none; border-radius: 5px; }
        </style>
      </head>
      <body>
        <div class="header">
          <img src="https://your-domain.com/logo.png" alt="Your Company" height="40">
          <h1>Welcome to Your Company</h1>
        </div>
        <div class="content">
          <h2>You're invited to join {{.OrganizationName}}!</h2>
          <p><strong>{{.AdminName}}</strong> has invited you to become <strong>{{.RoleName}}</strong> in the {{.OrganizationName}} organization.</p>
          <p>Once you accept, you'll be able to {{.RoleVerb}} the {{.OrganizationName}} organization.</p>
          <p><a href="https://your-domain.com/invite/{{.InvitationCode}}" class="button">Accept Invitation</a></p>
          <p>Or use the CLI: <code>minder auth invite accept {{.InvitationCode}}</code></p>
          <hr>
          <p><small>This invitation was sent to {{.RecipientEmail}}. If you weren't expecting this, you can ignore this email.</small></p>
          <p><small><a href="https://your-domain.com/terms">Terms</a> | <a href="https://your-domain.com/privacy">Privacy</a> | <a href="https://your-domain.com/login">Sign In</a></small></p>
        </div>
      </body>
    </html>
  invite-email.txt.tmpl: |
    You're invited to join {{.OrganizationName}}!
    
    {{.AdminName}} has invited you to become {{.RoleName}} in the {{.OrganizationName}} organization.
    
    Accept your invitation: https://your-domain.com/invite/{{.InvitationCode}}
    
    Or use the CLI:
    minder auth invite accept {{.InvitationCode}}
    
    If you are a member of multiple organizations, use:
    minder auth invite accept {{.InvitationCode}} --project {{.OrganizationId}}
    
    Once you accept, you'll be able to {{.RoleVerb}} the {{.OrganizationName}} organization.
    
    This invitation was sent to {{.RecipientEmail}}. If you weren't expecting this, you can ignore this email.
    
    Terms: https://your-domain.com/terms
    Privacy: https://your-domain.com/privacy
    Sign In: https://your-domain.com/login
    
    Your Company Team
```

### Mount in the Deployment

If you are managing your Kubernetes manifests directly, update your Minder `Deployment` to mount this `ConfigMap` using `subPath`. This ensures that only the specific template files are replaced, while the rest of the existing templates in `/var/run/ko/templates/` remain intact.

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: minder-server
spec:
  template:
    spec:
      containers:
        - name: minder-server
          image: ghcr.io/mindersec/minder:latest
          volumeMounts:
            - name: custom-template-volume
              mountPath: /var/run/ko/templates/invite-email.html.tmpl
              subPath: invite-email.html.tmpl
            - name: custom-template-volume
              mountPath: /var/run/ko/templates/invite-email.txt.tmpl
              subPath: invite-email.txt.tmpl
      volumes:
        - name: custom-template-volume
          configMap:
            name: minder-custom-email-template
```

### Reference via Helm Chart

If you used the Minder [Helm chart](https://docs.mindersec.dev/run_minder_server/installing_minder#helm-chart-parameters) to deploy Minder, you can configure these templates via the `deploymentSettings.extraVolumes` and `deploymentSettings.extraVolumeMounts` values.

```yaml
deploymentSettings:
  extraVolumes:
    - name: custom-template-volume
      configMap:
        name: minder-custom-email-template
  extraVolumeMounts:
    - name: custom-template-volume
      mountPath: /var/run/ko/templates/invite-email.html.tmpl
      subPath: invite-email.html.tmpl
    - name: custom-template-volume
      mountPath: /var/run/ko/templates/invite-email.txt.tmpl
      subPath: invite-email.txt.tmpl
```

## 2. The Modern Approach: Using OCI Volume Sources (Kubernetes v1.35+)

Kubernetes v1.35 introduces beta support for **OCI volume sources** (`image` volumes). This allows you to package your customized templates directly into a sterile OCI image and mount it over the directory, streamlining the customization workflow—especially if you have numerous templates. 

> [!NOTE]  
> The `ImageVolume` feature gate must be enabled in your Kubernetes v1.35+ cluster.

### Package your templates

Create a highly minimal Docker image that only contains your template files at the exact path you plan to mount or at its root:

```dockerfile
# Example Dockerfile for custom email templates
FROM scratch
COPY invite-email.html.tmpl /
COPY invite-email.txt.tmpl /
```

Build and push this artifact to a container registry:

```bash
docker build -t ghcr.io/your-org/minder-templates:v1 .
docker push ghcr.io/your-org/minder-templates:v1
```

### Mount the OCI Volume Source

Update the Minder `Deployment` to use the `image` volume type.

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: minder-server
spec:
  template:
    spec:
      containers:
        - name: minder-server
          image: ghcr.io/mindersec/minder:latest
          volumeMounts:
            - name: custom-templates
              mountPath: /var/run/ko/templates/invite-email.html.tmpl
              subPath: invite-email.html.tmpl
            - name: custom-templates
              mountPath: /var/run/ko/templates/invite-email.txt.tmpl
              subPath: invite-email.txt.tmpl
      volumes:
        - name: custom-templates
          image:
            reference: ghcr.io/your-org/minder-templates:v1
            pullPolicy: IfNotPresent
```

*By utilizing an image volume, administrators don't have to keep complex multi-line HTML templates inside Kubernetes ConfigMaps.*

## Debugging

Because Minder is built as a minimal image using `ko`, it does not contain a shell or basic utilities like `ls` or `cat`. Therefore, you cannot use `kubectl exec` to verify the templates at runtime.

To verify your custom volume mounts, inspect the configuration of your Pods in Kubernetes:

```bash
kubectl describe deployment minder-server
```

If you need to analyze the contents of the built image (e.g., specifically when using OCI volume sources), you can use an external tool like [dive](https://github.com/wagoodman/dive):

```bash
dive ghcr.io/your-org/minder-templates:v1
```
