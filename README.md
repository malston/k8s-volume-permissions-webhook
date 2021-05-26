# Kubernetes Mutating Webhook for Volume Permissions Init Container

My homelab datastore is backed by NFS which means that a container with a mounted persistence volume will only be able
to write to the backing datastore if it's running as root. The standard practice of deploying containers with non-root
privileges will result in those containers only having read-only access to the volume. Unfortunately, setting the
permissions on the mounted volume to the non-root user using `fsGroup` in a SecurityContext does not work. The
[workaround](https://docs.bitnami.com/tutorials/work-with-non-root-containers/) is to deploy an init container that runs
as the root user to modify the owner of the volume mount to the non-root user.

This mutating webhook will chown the volume mount whenever a pod is created with a container or init-container that
has a PodSecurityContext (or container SecurityContext) to the owner specified.

## Build

1. Build binary

```shell
make build
```

2. Build docker image

```shell
make build-image
```   

3. Push docker image

```shell
make push-image
```

> Note: log into the docker registry before pushing the image.

## Deploy

1. Create namespace `volume-permissions-container-injector` in which the webhook is deployed

```shell
kubectl create ns volume-permissions-container-injector
```

2. Create a signed cert/key pair and store it in a Kubernetes `secret` that will be consumed by sidecar injector deployment:

```shell
./deploy/webhook-create-signed-cert.sh \
    --service vpci-webhook-svc \
    --secret volume-permissions-container-injector-webhook-certs \
    --namespace volume-permissions-container-injector
```

3. Patch the `MutatingWebhookConfiguration` by set `caBundle` with correct value from Kubernetes cluster

```shell
cat deploy/mutatingwebhook.yaml | deploy/webhook-patch-ca-bundle.sh > deploy/mutatingwebhook-ca-bundle.yaml
```

4. Deploy resources

```shell
kubectl apply -f deploy/deployment.yaml
kubectl apply -f deploy/service.yaml
kubectl apply -f deploy/role.yaml
kubectl apply -f deploy/mutatingwebhook-ca-bundle.yaml
```

## Verify

1. The webhook should be in running state

```shell
kubectl -n volume-permissions-container-injector get pod
NAME                                                   READY   STATUS    RESTARTS   AGE
volume-permissions-container-injector-webhook-deployment-7c8bc5f4c9-28c84   1/1     Running   0          30s
```

```shell
kubectl -n volume-permissions-container-injector get deploy
NAME                                  READY   UP-TO-DATE   AVAILABLE   AGE
volume-permissions-container-injector-webhook-deployment   1/1     1            1           67s
```

2. Follow the logs of the webhook

```shell
kubectl -n volume-permissions-container-injector logs -l app=volume-permissions-container-injector --follow
```

3. Create new namespace `sentry-pro` and label it with `volume-permissions-container-injector=enabled`

```shell
kubectl create ns sentry-pro
kubectl label namespace sentry-pro volume-permissions-container-injection=enabled
kubectl get namespace -L volume-permissions-container-injection
NAME                                      STATUS   AGE   VOLUME-PERMISSIONS-CONTAINER-INJECTION
default                                   Active   26m
sentry-pro                                 Active   13s   enabled
kube-public                               Active   26m
kube-system                               Active   26m
volume-permissions-container-injector     Active   17m
```

4. Deploy an app in Kubernetes cluster, take `sentry-redis` statefulset as an example

```shell
kubectl apply -f examples/sentry-redis-sts.yaml -n sentry-pro
```

5. Verify init container is injected

```shell
kubectl get pod
NAME                    READY   STATUS     RESTARTS   AGE
sentry-redis-master-0   0/1     Init:0/1   0          12s
```

```shell
kubectl -n sentry-pro get pod sentry-redis-master-0 -o jsonpath="{.spec.initContainers[*].name}"
volume-permissions
```

# Cleanup

```shell
kubectl delete sts sentry-redis-master
kubectl delete cm sentry-redis-master-0-configmap
kubectl delete pvc redis-data-sentry-redis-master-0
```

## Troubleshooting

Sometimes you may find that pod is not injected with an init container as expected. Check the following items:

1. The volume-permissions-container-injector webhook is in running state and no error logs.
2. The namespace in which application pod is deployed has the correct labels as configured in `mutatingwebhookconfiguration`.
3. Check the `caBundle` is patched to `mutatingwebhookconfiguration` object by checking if `caBundle` fields is empty.
4. Check if the application pod has annotation `volume-permissions-container-injector-webhook.morven.me/inject":"yes"`.
5. Check if the configmap is there: `kubectl get cm sentry-redis-master-0-configmap -ojsonpath={.data.'volumepermissions\.yaml'}`
