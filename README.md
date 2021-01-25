# ViNO Cluster Operator

[![Docker Repository on Quay](https://quay.io/repository/airshipit/vino/status "Docker Repository on Quay")](https://quay.io/repository/airshipit/vino)

## Overview

The lifecycle of the Virtual Machines and their relationship to the Kubernetes cluster will be
managed using two operators: vNode-Operator(ViNO), and the Support Infra Provider Operator (SIP).


## Description

ViNO is responsible for setting up VM infrastructure, such as:

- per-node vino pod:
    * libvirt init, e.g.
        * setup vm-infra bridge
        * provisioning tftp/dhcp definition
    * libvirt launch
    * sushi pod
- libvirt domains
- networking
- bmh objects, with labels:
    * location - i.e. `rack: 8` and `node: rdm8r008c002` - should follow k8s semi-standard
    * vm role - i.e. `node-type: worker`
    * vm flavor - i.e `node-flavor: foobar`
    * networks - i.e. `networks: [foo, bar]`
      and the details for ViNO can be found [here](https://hackmd.io/KSu8p4QeTc2kXIjlrso2eA)

The Cluster Support Infrastructure Provider, or SIP, is responsible for the lifecycle of:
- identifying the correct `BareMetalHost` resources to label (or unlabel) based on scheduling
  constraints.
- extract IP address information from `BareMetalHost` objects to use in the creation of supporting
  infrastructure.
- creating support infra for the tenant k8s cluster:
    * load balancers (for tenant Kubernetes API)
    * jump pod to access the cluster and nodes via ssh
    * an OIDC provider for the tenant cluster, i.e. Dex
    * potentially more in the future

## Development Environment

### Pre-requisites

#### Install Golang 1.15+

ViNO is a project written in Go, and the make targets used to deploy ViNO leverage both Go and
Kustomize commands which require Golang be installed.

For detailed installation instructions, please see the [Golang installation guide](https://golang.org/doc/install).

#### Install Kustomize v3.2.3+

In order to apply manifests to your cluster via Make targets we suggest the use of Kustomize.

For detailed installation instructions, please see the [Kustomize installation guide](https://kubectl.docs.kubernetes.io/installation/kustomize/).

#### Proxy Setup

If your organization requires development behind a proxy server, you will need to define the
following environment variables with your organization's information:

```
HTTP_PROXY=http://username:password@host:port
HTTPS_PROXY=http://username:password@host:port
NO_PROXY="localhost,127.0.0.1,10.96.0.0/12"
PROXY=http://username:password@host:port
USE_PROXY=true
```

10.96.0.0/12 is the Kubernetes service CIDR.

### Deploy ViNO

Airship projects often have to deploy Kubernetes, with common requirements such as supporting
network policies or working behind corporate proxies. To that end the community maintains a
Kubernetes deployment script and is the suggested way of deploying your Kubernetes cluster for
development purposes.

#### Deploy Kubernetes

```
# curl -Lo deploy-k8s.sh https://opendev.org/airship/charts/raw/branch/master/tools/gate/deploy-k8s.sh
# chmod +x deploy-k8s.sh
# sudo ./deploy-k8s.sh
```

#### (Optional) Configure Docker to run as non-root

When Kubernetes is deployed from the script above it installs Docker but does not configure it
to run as a non-root user. The shell commands below are optional to configure Docker to run as
a non-root user. They include creating the docker group, adding the current user to that group
updating the group without having to log out and testing functionality with the hello-world
container.

If you choose to skip these steps, please continue with the developer environment steps as a
root user.

```
# sudo groupadd docker
# sudo usermod -aG docker $USER
```

Log out and log back in again for the changes to take effect, then test functionality with a
hello world container.

```
# docker run hello-world
```

#### Deploy ViNO

Once your cluster is up and running, you'll need to build the ViNO image to use, and to deploy the
operator on your cluster:

```
# make docker-build-controller
# make deploy
```

Once these steps are completed, you should have a working cluster with ViNO deployed on top of it:

```
# kubectl get pods --all-namespaces
NAMESPACE     NAME                                        READY   STATUS    RESTARTS   AGE
kube-system   calico-kube-controllers-7985fc4dd6-6q5l4    1/1     Running   0          3h7m
kube-system   calico-node-lqzxp                           1/1     Running   0          3h7m
kube-system   coredns-f9fd979d6-gbdzl                     1/1     Running   0          3h7m
kube-system   etcd-ubuntu-virtualbox                      1/1     Running   0          3h8m
kube-system   kube-apiserver-ubuntu-virtualbox            1/1     Running   0          3h8m
kube-system   kube-controller-manager-ubuntu-virtualbox   1/1     Running   0          3h8m
kube-system   kube-proxy-ml4gd                            1/1     Running   0          3h7m
kube-system   kube-scheduler-ubuntu-virtualbox            1/1     Running   0          3h8m
kube-system   storage-provisioner                         1/1     Running   0          3h8m
vino-system   vino-controller-manager-788b994c74-sbf26    2/2     Running   0          25m
```

#### Test basic functionality

```
# kubectl apply -f config/samples/vino_cr.yaml
# kubectl get pods
# kubectl get ds
```

delete vino CR and make sure DaemonSet is deleted as well

```
# kubectl delete vino vino-test-cr
# kubectl get ds
# kubectl get cm
```

## Get in Touch

For any questions on the ViNo, or other Airship projects, we encourage you to join the community on
Slack/IRC or by participating in the mailing list. Please see this [Wiki](https://wiki.openstack.org/wiki/Airship#Get_in_Touch) for
contact information, and the community meeting schedules.
