# ManifestUpdater

The ManifestUpdater is kubernetes operator to update image tags in manifest and create a PullRequest when a container image was pushed to docker registry.
This is one of the necessary components to follow the GitOps manner.

## Quick start

To install ManifestUpdater to your cluster.

```sh
kubectl apply -k deploy/
```

## CRD

The following sample custom resource is `Updater` object that creates a PullRequest when the new tag was pushed to specified dockehub registry.

```yaml
apiVersion: manifest-updater.koyuta.io/v1alpha1
kind: Updater
metadata:
  name: example-updater
spec:
  registry:
    dockerHub: index.docker.io/example/app
    filter: dev-*
  repository:
    git: https://github.com/koyuta/manifests
    branch: master
    path: /overlay/prd
```

Here is a spec field of `Updater`:

| Field | Key | Description |
|-------|-----|-------------|
|registry|dockerHub|The resource url of dockerhub.|
|registry|filter|Extract image tags matched by filter regexp. (Optional)|
|repository|git|The manifest repository url. Preferable to use https protocol.|
|repository|base|The base branch of PullRequest. (Optional, default: `master`)|
|repository|head|The head branch of PullRequest. (Optional, default: `feature/update-tag`)|
|repository|path|Rewrites only the tags below that path. (Optional, default: `/`)|


## Provide a github token

Since ManifestUpdater uses the github api to create PullRequest, you have to provide a your own github token.
The recommended approach is to provide github token as `Secret` value.

1. Create a secret resource:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: manifest-updater
type: Opaque
data:
  token: <base64 encoded github api token>
```

2. Provide a github token:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: manifest-updater-controller-manager
spec:
  template:
    spec:
      containers:
        - name: manager
          image: manifest-updater:latest
          args:
            - --metrics-addr=127.0.0.1:8080
            - --interval=12
            - --user=<github user name>
            - --token=$(TOKEN)
          env:
            - name: TOKEN
              valueFrom:
                secretKeyRef:
                  name: manifest-updater
                  key: token
```
