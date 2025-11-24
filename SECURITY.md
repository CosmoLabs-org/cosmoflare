# 🔐 Security Information

## Installation Security

Our installation scripts are designed with security as the top priority:

### ✅ What Our Installers Do
- **Download Only**: Fetch official releases from GitHub
- **No Elevated Privileges**: Install to user directory only
- **Transparent**: All actions are logged and visible
- **Open Source**: Scripts are fully auditable

### ❌ What Our Installers Never Do
- **No sudo/admin access**: Never request administrator privileges
- **No system modification**: Never modify system files
- **No data collection**: Never send any data anywhere
- **No background services**: Never install persistent services

## Manual Verification

You can always verify the installation:

```bash
# Verify the download source
curl -I https://github.com/CosmoLabs-org/CosmoDev-R2Go2/releases

# Check the binary integrity (after installation)
./r2go2 --version
```

## API Token Security

Your Cloudflare API tokens are valuable credentials:

### ✅ Secure Practices
- **Minimal Permissions**: Use only the permissions you need
- **Token Rotation**: Regularly rotate your API tokens
- **Secure Storage**: R2Go2 encrypts your configuration files
- **Environment Variables**: Safe for CI/CD automation

### 🔒 Recommended Token Permissions

For full R2 management:
- **Account**: `Cloudflare R2:Edit`

For read-only access:
- **Account**: `Cloudflare R2:Read`

### 🚫 Never Do This
- **Commit tokens to version control**
- **Share tokens publicly**
- **Use overly permissive tokens**
- **Store tokens in plain text files**

## Configuration File Security

R2Go2 stores your configuration securely:

```bash
# Configuration location
~/.config/r2go2/config.json

# The file is encrypted and permissions-restricted
chmod 600 ~/.config/r2go2/config.json
```

### Profile Security Features
- **Encrypted Storage**: Sensitive data is encrypted at rest
- **Permission Restrictions**: Files have appropriate Unix permissions
- **No Plain Text**: Tokens are never stored in plain text
- **Local Only**: Configuration never leaves your system

## Reporting Security Issues

Found a security vulnerability? Please report it responsibly:

- **Email**: security@cosmolabs.org
- **Private**: Please report privately, not in public issues
- **Response**: We'll respond within 24 hours

## Security Best Practices

### For Users
1. **Use API Tokens**: Prefer tokens over global API keys
2. **Least Privilege**: Grant only necessary permissions
3. **Regular Rotation**: Rotate tokens periodically
4. **Secure Networks**: Only use on trusted networks
5. **Keep Updated**: Use the latest version of R2Go2

### For Organizations
1. **Service Accounts**: Create dedicated service accounts
2. **IP Restrictions**: Restrict API tokens to specific IPs
3. **Audit Logs**: Monitor Cloudflare audit logs
4. **Token Policies**: Implement token lifecycle policies
5. **Team Training**: Train team on security best practices

## Compliance

R2Go2 is designed with compliance in mind:

- **GDPR Compliant**: No personal data collection
- **SOC 2 Ready**: Secure development practices
- **Enterprise Ready**: Suitable for enterprise environments
- **Open Source**: Fully auditable codebase

---

**Security is everyone's responsibility** 🛡️

If you have any security questions or concerns, please don't hesitate to reach out to our security team.