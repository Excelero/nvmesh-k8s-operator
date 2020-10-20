# Developer Guide

## Update version
In order to update the version info edit the file at `mainfests/config.yaml`
Then update all yaml manifests to the new version by running:
```bash
make manifests
```

## Build

### Build Binaries and Manifests
To build all binaries and yaml manifests, run:
```bash
make all
```

### Build Operator image
To build the operator docker image run:
```bash
make docker-build
```

## Test bundle on development cluster
Build the bundle image and index image
```bash
make bundle-registry-build
```
**NOTE**
This requires the [operator-registry](https://github.com/operator-framework/operator-registry) project to be cloned into your `~/home/projects` directory
```bash
cd $HOME/projects
git clone git@github.com:operator-framework/operator-registry.git
```

**NOTE**
The command will try to push the bundle images to DockerHub.com

## Deploying to RedHat OpenShift OperatorHub
Build the bundle image for deployment in *RedHat OpenShift OperatorHub*:
```bash
make bundle-build
```

To upload the image follow the instructions on the RedHat Partner Connect website for uploda, scan and certification


