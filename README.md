# External Ambassador

Utility that runs inside of the Kubernetes cluster that takes the host entries
on ambassador mappings defined in service annotations and converts them to
external-dns annotations on the ambassador service.

## Dependencies

This requires a deployed Ambassador (getambassador.io) and External DNS (github.com/kubernetes-incubator/external-dns). Once both of those projects are deployed in the cluster this utility will automate the syncing of DNS to point at the ambassador service.

## Deploy in Kubernetes

This just runs as a simple Deployment in the cluster. It will automatically watch for changes in the cluster and make the required adjustments so external-dns will pick up the changes.

All the examples assume that ambassador and external-dns are running the same namespace "ambassador".

### With RBAC (most common)

```
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: external-ambassador
  namespace: ambassador
---
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRole
metadata:
  name: external-ambassador
rules:
- apiGroups: [""]
  resources: ["services"]
  verbs: ["get","watch","list"]
---
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRoleBinding
metadata:
  name: external-ambassador-viewer
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: external-ambassador
subjects:
- kind: ServiceAccount
  name: external-ambassador
  namespace: ambassador
---
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: external-ambassador
  namespace: ambassador
spec:
  strategy:
    type: Recreate
  template:
    metadata:
      labels:
        app: external-ambassador
    spec:
      serviceAccountName: external-ambassador
      containers:
      - name: external-dns
        image: blakerouse/external-ambassador:latest
        args:
        - --namespace=ambassador
```

### Without RBAC

```
---
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: external-ambassador
  namespace: ambassador
spec:
  strategy:
    type: Recreate
  template:
    metadata:
      labels:
        app: external-ambassador
    spec:
      containers:
      - name: external-dns
        image: blakerouse/external-ambassador:latest
        args:
        - --namespace=ambassador
```
