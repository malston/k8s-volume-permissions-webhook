# Kubernetes Mutating Webhook for Volume Permissions Init Container

## Prerequisites

- [git](https://git-scm.com/downloads)
- [go](https://golang.org/dl/) version v1.12+
- [docker](https://docs.docker.com/install/) version 17.03+
- [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/) version v1.11.3+
- Access to a Kubernetes v1.11.3+ cluster with the `admissionregistration.k8s.io/v1beta1` API enabled. Verify that by the following command:

```shell
kubectl api-versions | grep admissionregistration.k8s.io
```

The result should be:

```shell
admissionregistration.k8s.io/v1
admissionregistration.k8s.io/v1beta1
```

> Note: In addition, the `MutatingAdmissionWebhook` and `ValidatingAdmissionWebhook` admission controllers should be added and listed in the correct order in the admission-control flag of kube-apiserver.

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
    --service vpci-svc \
    --secret volume-permissions-container-injector-webhook-certs \
    --namespace volume-permissions-container-injector
```

3. Patch the `MutatingWebhookConfiguration` by set `caBundle` with correct value from Kubernetes cluster

```shell
cat deploy/mutatingwebhook.yaml | \
    deploy/webhook-patch-ca-bundle.sh > \
    deploy/mutatingwebhook-ca-bundle.yaml
```

4. Deploy resources

```shell
kubectl create -f deploy/deployment.yaml
kubectl create -f deploy/service.yaml
kubectl create -f deploy/mutatingwebhook-ca-bundle.yaml
```

## Verify

1. The webhook should be in running state

```shell
kubectl -n volume-permissions-container-injector get pod
NAME                                                   READY   STATUS    RESTARTS   AGE
volume-permissions-container-injector-webhook-deployment-7c8bc5f4c9-28c84   1/1     Running   0          30s

kubectl -n volume-permissions-container-injector get deploy
NAME                                  READY   UP-TO-DATE   AVAILABLE   AGE
volume-permissions-container-injector-webhook-deployment   1/1     1            1           67s
```

2. Create new namespace `injection` and label it with `volume-permissions-container-injector=enabled`

```shell
kubectl create ns injection
kubectl label namespace injection volume-permissions-container-injection=enabled
kubectl get namespace -L volume-permissions-container-injection

NAME                                      STATUS   AGE   VOLUME-PERMISSIONS-CONTAINER-INJECTION
default                                   Active   26m
injection                                 Active   13s   enabled
kube-public                               Active   26m
kube-system                               Active   26m
volume-permissions-container-injector     Active   17m
```

3. Deploy an app in Kubernetes cluster, take `alpine` app as an example

```shell
kubectl run alpine --image=alpine --restart=Never -n injection --overrides='{"apiVersion":"v1","metadata":{"annotations":{"volume-permissions-container-injector-webhook.malston.me/inject":"yes"}}}' --command -- sleep infinity
```

4. Verify sidecar container is injected

```shell
kubectl get pod
NAME                     READY     STATUS        RESTARTS   AGE
alpine                   2/2       Running       0          1m
```

```shell
kubectl -n injection get pod alpine -o jsonpath="{.spec.containers[*].name}"
alpine sidecar-nginx
```

## Troubleshooting

Sometimes you may find that pod is not injected with an init container as expected. Check the following items:

1. The volume-permissions-container-injector webhook is in running state and no error logs.
2. The namespace in which application pod is deployed has the correct labels as configured in `mutatingwebhookconfiguration`.
3. Check the `caBundle` is patched to `mutatingwebhookconfiguration` object by checking if `caBundle` fields is empty.
4. Check if the application pod has annotation `volume-permissions-container-injector-webhook.morven.me/inject":"yes"`.
