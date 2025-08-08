# MCP Server Operator

The **MCP server Operator** manages the lifecycle of MCP (Model Context protocol) Servers on OpenShift Clusters. It leverages a Custom Resource definition (CRD) Called `MCPServer`, allowing users to specify the MCP server image and runtime arguments for a custom MCP server deployment.

## Features
- Deploy and manages MCP server instances via CRDs
- Supports custom container images and runtime arguments
- Compatible with Openshift clusters
- Includes both end-to-end test and unit tests.
## Table of Contents
- [Project Overview](#mcp-server-operator)
- [Features](#features)
- [Usage](#usage)
    - [Prerequisites](#prerequisites)
    - [Installation](#Installation)
    - [Running the operator locally](#Running-The-Operator-Locally)
    - [Running the operator on a cluster](#Running-the-operator-on-a-cluster)
    - [Making an MCP Server Instance](#making-an-mcp-server-instance)
    - [Uninstalling the operator and cleaning the cluster](#Uninstalling-the-operator-and-cleaning-the-cluster)
- [Developer Guide](#Developer-Guide)
  - [Pre-requisites](#Pre-requisites)
  - [Run tests](#Run-tests)
  - [Contributing](#Contributing)


## Usage

### Prerequisites

Before installation, you will need the following to run the operator.

- An **OpenShift Cluster** (ROSA, OSD, CRC) or compatible Kubernetes cluster
- A **container engine** (`podman` or `docker`)
- The OpenShift CLI tool: `oc`
- Sufficient permissions to install CRDs and deploy operators

### Installation

Before doing any of these steps, ensure that you are logged into your cluster if using one.

```
oc login --token=<your user token> --server=<your openshift cluster server>
```


After logging into your cluster, clone the repository and then swap to the mcp-server-operator folder.
```
git clone https://github.com/opendatahub-io/mcp-server-operator.git
```
```
cd mcp-server-operator
```
Next, set the environment variables:
```
export IMG=<your-registry>/<username>/mcp-server-operator:<tag>
```


Afterward, build the image.
```
CONTAINER_TOOL={docker|podman} IMG=$IMG make build
```



Then, install the necessary CRDs.

```
make install
```

### Running The Operator Locally

```
make run
```

### Running The Operator on a Cluster

```
make deploy IMG=$IMG
```

### Making an MCP Server Instance

The following is an example on how to create an MCPServer, ensure that the text in brackets is replaced with the appropriate information before running the command.

```
cat << EOF | oc create -f -
apiVersion: mcpserver.opendatahub.io/v1
kind: MCPServer
metadata:
  name: <your_name_here>
  namespace: mcp-server-operator-system
spec:
  image: <your_image_here>
EOF
```

**Field Descriptions**
- `image`: Container image for the MCP server.
- `args`: (Optional) List of arguments passed to the MCP server container.

### Uninstalling the operator and cleaning the cluster
Firstly, delete the MCPServer object from the cluster with using the following:

```
oc delete mcpserver -n mcp-server-operator-system [name]
```
Next is to uninstall the operator, which can be done by simply running the command below.
```
make undeploy
```


## Developer Guide

### Pre-requisites
* `Go 1.23`
* `Kubebuilder`

### Run Tests

To run unit tests, run

```
make unit-test
```

To run end-to-end tests, run

```
make-e2e
```

### Contributing

Contributions are welcome! Please refer to our [contributing guidelines](https://github.com/opendatahub-io/opendatahub-community/blob/main/contributing.md).


