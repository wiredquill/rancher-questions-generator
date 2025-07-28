# Rancher Questions Generator

A professional-grade web application for creating `questions.yaml` files for Helm charts with an intuitive drag-and-drop interface. Built specifically for Rancher environments and Helm chart authoring workflows.

## ✨ Features

- **🔍 Repository Management**: Browse and search Helm chart repositories
- **📦 Chart Processing**: Extract and analyze values.yaml from Helm charts
- **🛠️ Visual Form Builder**: Drag-and-drop interface for creating questions
- **⚡ Quick Add Templates**: Pre-built question templates for common scenarios (Service, Storage, Ingress, Resources)
- **👀 Live Preview**: Real-time preview of the generated form
- **💾 Project Management**: Save and load your work
- **🎯 Smart Defaults**: Automatically detect common patterns and suggest question types
- **🔄 Values Explorer**: Tree and raw YAML view of chart values
- **🌐 Repository Browsing**: Browse charts from popular repositories including Rancher Partner Charts
- **🔐 Authentication Support**: Support for protected OCI registries like SUSE Application Collection

## 🚀 Quick Start

### Prerequisites

- Kubernetes cluster with kubectl access
- Container runtime (for building images)

### Deployment

1. **Clone the repository**:
```bash
git clone https://github.com/wiredquill/rancher-questions-generator.git
cd rancher-questions-generator
```

2. **Build and deploy**:
```bash
# Apply Kubernetes manifests
kubectl apply -f k8s/
```

3. **Access the application**:
The application will be available at the LoadBalancer IP (check with `kubectl get svc`).

### Using Helm Chart (Recommended)

```bash
# Add the repository
helm repo add rancher-questions-generator https://wiredquill.github.io/rancher-questions-generator

# Install the chart
helm install my-questions-generator rancher-questions-generator/rancher-questions-generator
```

Input: The user provides a URL to a Helm chart .tgz file.

Processing:

The backend downloads the specified Helm chart archive.

It unpacks the .tgz file in a temporary, isolated directory.

It parses the values.yaml to identify all configurable parameters.

It looks for an existing questions.yaml. If one exists, it's parsed; if not, a default structure is created.

Form Emulation:

The backend serves the parsed values.yaml data and the questions.yaml structure to the frontend.

The frontend dynamically renders an interactive form based on this data.

Drag-and-Drop Interface:

A two-panel UI will be presented.

Panel 1 (Values Explorer): Displays a tree-view of all keys from the values.yaml. Users can browse all available options.

Panel 2 (Form Builder): Represents the questions.yaml. Users can drag keys from the Values Explorer into this panel to "expose" them as questions for the end-user.

Default Questions: The Form Builder will be pre-populated with a set of default, essential questions:

Application Name: variable: name

Namespace: variable: namespace

Storage Class: variable: persistence.storageClass (with logic to detect if persistence is used)

Service Type: variable: service.type (presented as a dropdown with options like ClusterIP, NodePort, LoadBalancer)

Configuration & Tweaking:

Once a value is dragged into the Form Builder, the user can configure its properties as defined by the questions.yaml spec.

Helper Text: An input field to add description text for each question.

Labels & Grouping: Ability to set the label and group for organization.

Type Casting: Define the input type (e.g., string, int, boolean, enum).

Live Preview & Output:

As the user makes changes in the Form Builder, the changes are reflected in real-time.

A "Get questions.yaml" button will allow the user to copy or download the generated YAML file.

3. Backend API Endpoints
A RESTful API will facilitate communication between the frontend and backend.

Method	Endpoint	Description
POST	/api/chart	Accepts a JSON payload like { "url": "..." }. Downloads and processes the chart. Returns a session ID.
GET	/api/chart/{session_id}	Retrieves the parsed values.yaml and questions.yaml for the given session.
PUT	/api/chart/{session_id}	Updates the questions.yaml structure for the session based on user changes in the UI.
GET	/api/chart/{session_id}/q	Returns the raw, generated questions.yaml file for the current state.

Export to Sheets
4. Technology Stack Suggestion
This stack is chosen for its robustness, performance, and compatibility with a cloud-native environment.

Frontend:

Framework: React with TypeScript for type safety.

UI Components: Material-UI (MUI) or Ant Design for pre-built components.

Drag & Drop: react-dnd or dnd-kit.

State Management: Zustand or Redux Toolkit.

Backend:

Language: Go is an excellent choice due to its performance, static typing, and strong support for concurrency, which is ideal for handling downloads and file processing. The standard library is well-suited for creating an HTTP server.

Libraries:

helm.sh/helm/v3/pkg/chart/loader for loading Helm charts.

gopkg.in/yaml.v3 for parsing YAML files.

Deployment:

Containerization: Docker.

Orchestration: Kubernetes.

CI/CD: GitHub Actions or GitLab CI to build and push container images and deploy to the cluster.







