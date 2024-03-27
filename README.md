# Virtualization CSI Driver

This repository hosts the Virtualization CSI driver and all of its build and dependent 
configuration files to deploy the driver.

This CSI driver is made for a guest cluster deployed on top of deckhouse virtual machines, 
and enables it to get its persistent data from the underlying, host cluster.
To avoid confusion, this CSI driver is deployed on the guest cluster, and 
does not require deckhouse installation at all.

The term "guest cluster" refers to the k8s cluster installed on deckhouse virtual machines, and "host cluster" refers 
to a cluster with deckhouse installed.

// TODO design image.

## Pre-requisite

- Kubernetes host cluster
  - kubectl
  - jq
- Kubernetes guest cluster
  - helm

## Deployment

### In host cluster

1. Modify `deploy/host/kustomization.yaml` by setting the _namespace_ according to the virtual machines 
on which the guest cluster is deployed:
```deploy/host/kustomization.yaml
namespace: default
resources:
- rbac.yaml
```

2. Create service account with roles required for Virtualization CSI Driver:
```shell
kubectl apply -k deploy/host/
```

3. Create base64 of host cluster kubeconfig for the CSI service account:
pass the cluster server value using the internal ip (from `kubectl get endpoints kubernetes`):
```shell
# ! Change SERVER_WITH_INTERNAL_IP value below ! 
SERVER_WITH_INTERNAL_IP=https://172.18.18.50:6443
curl -s https://raw.githubusercontent.com/deckhouse/virtualization-csi-driver/main/scripts/get-host-kubeconfig.sh | bash /dev/stdin --server $SERVER_WITH_INTERNAL_IP
```

### In guest cluster

1. Create `values.yaml` file in _deploy/guest/values.yaml_ directory with the following parameters:
```deploy/guest/values.yaml
host:
    # namespace of deckhouse virtual machines in host cluster
    virtualMachineNamespace: default
    # base64 of host cluster kubeconfig for the CSI service account
    kubeconfig: XXXX=
guest:
    # namespace of csi driver in guest cluster
    csiDriverNamespace: default
```

2. Install Virtualization CSI Driver to guest cluster in the root of the repo:
```shell
helm install csi deploy/guest/
```

## Examples 

Examples of pvc are represented in _examples_ directory.

## Restrictions

Deckhouse Virtualization doesn't have ability to hotplug one disk to several virtual machines.
Thus, PVCs with access mode ReadWriteMany or ReadOnlyMany currently aren't supported by Virtualization CSI Driver.

## Useful tasks

- `push` — build csi driver and push to dev-registry.deckhouse.io
- `lint` — run linters
