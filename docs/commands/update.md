---
layout: default
title: Update
parent: Commands
nav_order: 5
---

# Update

Updates `gvm` to the latest version available on GitHub Releases.

## Usage

```bash
gvm update
```

## Description

The `update` command checks the GitHub Releases API for the latest version of `gvm`. If a newer version is available, it downloads the appropriate binary for your operating system and architecture, verifies its checksum, and replaces the current installation.

## Example

```bash
$ gvm update
Checking for updates...
New version available: v1.0.4 (current: v1.0.3)
Downloading update...
Verifying checksum...
Checksum verified.
Successfully updated to v1.0.4
```
