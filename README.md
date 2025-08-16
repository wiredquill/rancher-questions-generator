# Rancher Questions Generator - Helm Repository

This is the Helm repository for the Rancher Questions Generator chart.

## Usage

Add this repository to Helm:

```bash
helm repo add rancher-questions-generator https://wiredquill.github.io/rancher-questions-generator/
helm repo update
```

Install the chart:

```bash
helm install my-questions-generator rancher-questions-generator/rancher-questions-generator
```

## Rancher Installation

For Rancher Apps & Marketplace, add this repository:
- Name: `Rancher Questions Generator`
- Repository URL: `https://github.com/wiredquill/rancher-questions-generator`
- Branch: `main`