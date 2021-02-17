# Development devfile plugin registry

The purpose of this repo is as a short-term solution for enabling the testing of plugin resolution in DevWorkspaces (see: https://github.com/devfile/devworkspace-operator) when those plugins are specified by plugin ID and registry URL.

This repo consists of
1. A small go program that converts plugin meta.yamls (from https://github.com/eclipse/che-plugin-registry) to devfiles (see https://github.com/devfile/api)
2. A dockerfile that extracts meta.yamls from an existing che-plugin-registry and converts its meta.yamls into devfiles using the program above, and builds a new registry that serves devfiles instead.
3. A script that can be used to deploy the devfile plugin registry to an OpenShift or minikube cluster

## Limitations
- To avoid significant complexity, the conversion program cannot convert plugins that use init containers (i.e. Theia). Directory `/manual` has plugin devfiles that are copied into the resulting image.
- The DevWorkspaceOperator requires a slightly different che-machine-exec plugin, as the default listens only on localhost and the DevWorkspaceOperator does not deploy an in-container proxy yet.
- Plugins that require downloading `.vsix` files are implemented by having a `vsx-installer` component in the theia plugin that downloads `vsix`es from annotations on containers. **This is a *temporary* solution**. It also means that non-sidecar plugins are currently unsupported, as there's nowhere to define such an annotation.

This repo is intended for testing and development purposes only. 

## Building/Deploying
The `Dockerfile` at the root of the repo will 1) build the go binary, 2) grab meta.yamls from the current che-plugin-registry, and 3) build a new registry using the converted devfiles:
```bash
docker build -t my-plugin-registry .
```

To deploy the registry, use the `./deploy-registry.sh` script at the root of this repo. The script supports environment variables:
- `NAMESPACE` - namespace in which to deploy the registry, default `devworkspace-plugins`
- `REGISTRY_IMAGE` - the image to deploy for the regsitry, default `docker.io/amisevsk/devworkspace-plugin-registry:dev`
- `ROUTING_SUFFIX` - Suffix to use for generating plugin registry URL on Kubernetes (i.e. registry is deployed at `che-plugin-registry.${ROUTING_SUFFIX}`). 

    Required on Kubernetes; should be auto-detected on `minikube` (`$(minikube ip).nip.io`). Ignored when running on OpenShift as a route is created instead.
