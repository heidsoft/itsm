# Security Policy

## Reporting a Vulnerability

We take security vulnerabilities seriously. If you discover a security issue, please report it responsibly.

### How to Report

1. **Do NOT** create a public GitHub issue for security vulnerabilities
2. Email the maintainers directly at: (replace with your security contact email)
3. Include the following information:
   - Description of the vulnerability
   - Steps to reproduce
   - Potential impact
   - Any suggested fixes (optional)

### What to Expect

- Acknowledgment within 48 hours
- Regular updates on our progress
- Public disclosure after the fix is released
- Credit in the release notes (if you wish)

### Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 1.x.x   | :white_check_mark: |

### Security Best Practices for Deployers

- Always change default passwords (`admin123`)
- Use strong JWT secrets (minimum 32 characters)
- Enable HTTPS in production
- Restrict CORS origins (`ITSM_ALLOW_ALL_ORIGINS=false`)
- Keep dependencies updated (Dependabot is enabled)
- Rotate API keys regularly
- Follow the principle of least privilege for database access

### Vulnerability Disclosure Policy

We follow a [Coordinated Vulnerability Disclosure](https://en.wikipedia.org/wiki/Coordinated_vulnerability_disclosure) model:
1. We work with researchers to understand and confirm the issue
2. We develop and test a fix
3. We release the fix in a new version
4. We publish a security advisory on GitHub

For general security questions, please use [GitHub Discussions](https://github.com/heidsoft/itsm/discussions).