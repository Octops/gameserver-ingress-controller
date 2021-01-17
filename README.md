# Gameserver Ingress Controller
Automatic Ingress configuration for Game Servers managed by [Agones](https://agones.dev/site/).

The Gameserver Ingress Controller leverages the power of the [Kubernetes Ingress Controller](https://kubernetes.io/docs/concepts/services-networking/ingress-controllers/) to bring inbound traffic to dedicated game servers.

Players will be able to reach out to a dedicated game server using a custom domain and a secure connection. I.e.:`https://octops-2dnqv-jmqgp.mygame.com`
## Supported Agones Resources
- Fleets
- Stand-Alone GameServers

## How it works
When a gameserver is created by Agones, either as part of a Fleet or a stand-alone deployment, the gameserver ingress controller will handle the provisioning of a couple of resources.

It will use the information present in the gameserver annotations and metadata to create the required Ingress and dependencies.

As an example, a Fleet that looks like:
```yaml
apiVersion: "agones.dev/v1"
kind: Fleet
metadata:
  name: octops
  labels:
    cluster: gke-1.17
    region: us-east-1
spec:
  replicas: 3
  template:
    metadata:
      labels:
        cluster: gke-1.17
        region: us-east-1
      annotations:
        # Required annotation used by the controller
        octops.io/gameserver-ingress-domain: "mygame.com"
        octops.io/terminate-tls: "true"
        octops.io/issuer-tls-name: "letsencrypt-prod"
 ...
```

Deployed Gameservers:
```bash
NAME                 STATE   ADDRESS         PORT   NODE     AGE
octops-2dnqv-jmqgp   Ready   36.23.134.23    7437   node-1   10m
octops-2dnqv-d9nxd   Ready   36.23.134.23    7323   node-1   10m
octops-2dnqv-fr8tx   Ready   212.76.142.33   7779   node-2   10m
```

Ingresses created by the controller:
```bash
NAME                 HOSTS                           ADDRESS         PORTS     AGE
octops-2dnqv-jmqgp   octops-2dnqv-jmqgp.mygame.com                   80, 443   4m48s
octops-2dnqv-d9nxd   octops-2dnqv-d9nxd.mygame.com                   80, 443   4m46s
octops-2dnqv-fr8tx   octops-2dnqv-fr8tx.mygame.com                   80, 443   4m45s
```

List of Ingresses and Backends
```bash

https://octops-2dnqv-jmqgp.mygame.com/ ⇢ octops-2dnqv-jmqgp:7437
https://octops-2dnqv-d9nxd.mygame.com/ ⇢ octops-2dnqv-d9nxd:7323
https://octops-2dnqv-fr8tx.mygame.com/ ⇢ octops-2dnqv-fr8tx:7779
```

## Conventions
The table below shows how the information from the gameserver is used to compose the ingress settings.

| Gameserver                          | Ingress       |
| ----------------------------------- |:-------------:|
| name                                | hostname      |
| annotation: octops.io/gameserver-ingress-domain | domain |
|annotation: octops.io/terminate-tls | terminate TLS |
|annotation: octops.io/issuer-tls-name| name of the issuer |


## Requirements
The following components must be present on the Kubernetes cluster where the dedicated gameservers, and the controller will he hosted/deployed.

- [Agones](https://agones.dev/site)
  - https://agones.dev/site/docs/installation/install-agones/helm/
- [NGINX Ingress Controller](https://kubernetes.github.io/ingress-nginx/)
  - Choose the appropriate setup depending on your environment, network topology and cloud provider. It will affect how the Ingress Service will be exposed to the internet.
- [Cert-Manager](https://cert-manager.io/)
  - Make sure you have an `Issuer` that uses LetsEncrypt.
  
## Fleet and GameServer Manifests
The same configuration works for Fleets and GameServers. Add the following annotations to your manifest:
```yaml
annotations:
  octops.io/gameserver-ingress-domain: "mygame.com"
  octops.io/terminate-tls: "true"
  octops.io/issuer-tls-name: "selfsigned-issuer"
```

- **octops.io/gameserver-ingress-domain:** name of the domain to be used when creating the ingress. This is the public domain that players will use to reach out to the dedicated game server.
- **octops.io/terminate-tls:** it determines if the ingress will terminate TLS. If set to "false" it means that TLS will be terminated at the loadbalancer. In this case there won't be a certificated issued by the local cert-manager.
- **octops.io/issuer-tls-name:** required if `terminate-tls=true`. This is the name of the issuer that cert-manager will use when creating the certificate for the ingress.

## Deploy the Gameserver Ingress Controller

Deploy the controller by running:
```bash
$ kubectl apply -f deploy/install.yaml
```

Check the deployment:
```bash
$ kubectl -n octops-system get pods

# Expected output
NAME                                         READY   STATUS    RESTARTS   AGE
octops-ingress-controller-6b8dc49fb9-vr5lz   1/1     Running   0          3h6m
```

Check logs:
```bash
$ kubectl -n octops-system logs -f $(kubectl -n octops-system get pod -l app=octops-ingress-controller -o=jsonpath='{.items[*].metadata.name}')
```
