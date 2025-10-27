# picosend [![build](https://github.com/bsv9/picosend/actions/workflows/ci.yml/badge.svg)](https://github.com/bsv9/picosend/actions/workflows/ci.yml)&nbsp;[![Coverage Status](https://coveralls.io/repos/github/bsv9/picosend/badge.svg?branch=main)](https://coveralls.io/github/bsv9/picosend?branch=main)


A minimalistic application for sharing secrets securely with one-time access.

![Picosend Screenshot](static/images/picosend.png)

## Features

- **One-time secret sharing** - Secrets are automatically deleted after being read once
- **Configurable lifetime** - Set secrets to expire after 5 minutes, 1 hour, or 1 day
- **End-to-end encryption** - AES-256-CBC encryption with client-side encryption
- **QR code generation** - Automatically generates QR codes for easy mobile sharing
- **No persistent storage** - Secrets stored only in memory
- **No user accounts required** - Anonymous and hassle-free sharing
- **Self-hostable** - Deploy on your own infrastructure
- **Open source** - Transparent and auditable code
- **Robot protection** - Seamless verification code system
- **Transport security** - Prevents exposure through proxies or logs
- **Minimalistic design** - Simple and intuitive user interface
- **Secure Password Generation** - Generates strong, random passwords for enhanced security

## Quick Start


### Using Docker

```bash
# Pull the image
docker pull docker.io/bsv9/picosend

# Run with default settings
docker run -p 8080:8080 -d docker.io/bsv9/picosend

# Access picosend
open http://localhost:8080
```

> **Note**: The Docker image supports multiple architectures including amd64, arm64 allowing it to run on a variety of platforms including Raspberry Pi, AWS Graviton instances, and Apple Silicon devices.

### Manual Installation

```bash
# Clone the repository
git clone https://github.com/bsv9/picosend.git
cd picosend

# Build the application
go build -o picosend

# Run with default settings
./picosend

```

## Security Features

- **AES-256-CBC encryption** with PKCS7 padding
- **Client-side encryption** before transmission
- **Automatic secret deletion** after first retrieval
- **Time-based expiration** ensures secrets are deleted even if not accessed
- **Background cleanup** removes expired secrets from memory
- **Transport security** prevents proxy/logging exposure
- **Memory is securely wiped after secret deletion**

## License

MIT
