---
title: Build from Source
weight: 4
---

# Build from Source

Build and run JVP from source code.

## Prerequisites

- Go 1.21+
- Node.js 18+
- Task (task runner)

## Step 1: Clone the Repository

```bash
git clone https://github.com/jimyag/jvp.git
cd jvp
```

## Step 2: Build the Project

```bash
# Build complete binary file including frontend
task build
```

## Step 3: Run the Service

```bash
# Run JVP service (default port 7777)
./bin/jvp
```

## Step 4: Access Web Interface

After building, the frontend is embedded in the binary file. Access:

```
http://localhost:7777
```

## Local Debugging with Docker

```bash
# Build local debug image
task debug-image

# Modify image in docker-compose.yml to jvp:local, then start
docker compose up -d
```
