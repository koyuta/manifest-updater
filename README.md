# ManifestUpdater

The ManifestUpdater is kubernetes operator to update image tags in manifest and create a PullRequest when a container image was pushed to docker registry.
This is one of the necessary components to follow the GitOps manner.

## Getting started

To install ManifestUpdater to your cluster.

```
kubectl apply -k deploy/
```

## CRD

Following custom resource is create updater object that creates a PullRequest when the new tag was pushed to specified dockehub registry.

```
apiVersion: manifest-updater.koyuta.io/v1alpha1
kind: Updater
metadata:
  name: example-updater
spec:
  registry:
    dockerHub: index.docker.io/example/app
    filter: dev-*
  repository:
    git: git@github.com:koyuta/manifests
    branch: master
    path: /overlay/prd
```

### Features

Required:

|Name|Description|
|:--:|:---------:|
|registry.dockerHub|The resource url of dockerhub.|
|repository.git|Manifest repository.|

Options

| Name |Default|Description|
|:----:|:-----:|:---------:|
|filter|(Without filtering)|Extract image tags matched by filter regexp.|
|branch|`master`|The base branch of PullRequest.|
| path |`/`|Rewrites only the tags below that path.|
