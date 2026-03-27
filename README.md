# Octops Game Server Ingress Controller
Automatic Ingress configuration for Game Servers managed by [Agones](https://agones.dev/site/).

The Octops Controller leverages the power of the [Kubernetes Ingress Controller](https://kubernetes.io/docs/concepts/services-networking/ingress-controllers/) to bring inbound traffic to dedicated game servers.

Players will be able to connect to a dedicated game server using a custom domain and a secure connection. 

## Supported Agones Resources
- Fleets
- Stand-Alone GameServers

## Use Cases
- Real-time games using websocket

## Known Limitations
For the Octops Controller to work, an Ingress Controller must be present in the cluster. The one that has been mostly adopted by the Kubernetes community is the NGINX Ingress Controller. However, it has been reported by the Agones' community that for games based on websocket the NGINX controller might not be a good fit due to the lost of connections between restarts. Check https://kubernetes.github.io/ingress-nginx/how-it-works/#when-a-reload-is-required for details.

You can find more information on the original reported issue https://github.com/Octops/gameserver-ingress-controller/issues/21.

The connection drop behaviour is also present on alternatives like the [HAProxy Ingress Controller](https://github.com/haproxytech/kubernetes-ingress).

For that reasons the suggested Ingress Controller is the [Contour Ingress Controller](https://projectcontour.io/). The controller is built on top of the https://www.envoyproxy.io/ service proxy. Envoy can handle flawlessly updates while game servers and ingress resources are reconciled by the Octops Controller. 

## Requirements
The following components must be present on the Kubernetes cluster where the dedicated game servers, and the controller will be hosted/deployed.

- [Agones](https://agones.dev/site)
  - https://agones.dev/site/docs/installation/install-agones/helm/
- [Contour](https://projectcontour.io/) — install the variant that matches the backend you intend to use. Both can run side-by-side in the same cluster.

  **Ingress backend** (default — `octops.io/router-backend` absent or set to `ingress`):
  ```bash
  kubectl apply -f https://projectcontour.io/quickstart/contour.yaml
  ```
  This installs the standard Contour + Envoy deployment in the `projectcontour` namespace. It handles `networking.k8s.io/v1 Ingress` resources.

  **Gateway API backend** (`octops.io/router-backend: gateway`):
  ```bash
  # 1. Install Gateway API CRDs (standard channel)
  kubectl apply -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.5.1/standard-install.yaml

  # 2. Install the Contour Gateway Provisioner
  kubectl apply -f https://projectcontour.io/quickstart/contour-gateway-provisioner.yaml
  ```
  The Gateway Provisioner watches for `Gateway` resources and automatically creates a dedicated Envoy deployment per Gateway. No additional configuration is needed.

  After installing the provisioner, create a `GatewayClass`:
  ```bash
  kubectl apply -f deploy/gateway/gatewayclass.yaml
  ```

  Both installations are independent. Running the standard Contour install for Ingress and the Gateway Provisioner for Gateway API in the same cluster is fully supported.

  For either variant, update your DNS with a `*` wildcard record pointing to the load balancer address shown under `EXTERNAL-IP` in `kubectl -n projectcontour get svc`.
- [Cert-Manager](https://cert-manager.io/docs/) - [optional if you are managing your own certificates]
  - Check https://cert-manager.io/docs/tutorials/acme/http-validation/ to understand which type of ClusterIssuer you should use.
  - Make sure you have an `ClusterIssuer` that uses LetsEncrypt. You can find some examples on [deploy/cert-manager](deploy/cert-manager).
  - The name of the `ClusterIssuer` must be the same used on the Fleet annotation `octops.io/issuer-tls-name`.
  - Install (**Check for newer versions**): ```$ kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.20.0/cert-manager.yaml```

# Configuration and Manifests

## Ingress Routing Mode
The Octops controller supports 2 different types of ingress routing mode: Domain and Path.

This configuration is used by the controller when creating the ingress resource within the Kubernetes cluster.

Routing Mode is a Fleet or GameServer scoped configuration. A Fleet manifest defines the routing mode to all of its GameServers. For stand-alone GameServers, the routing mode is defined on its own manifest.

### Domain
Every game server gets its own [FQDN](https://en.wikipedia.org/wiki/Fully_qualified_domain_name#Example). I.e.:`https://octops-2dnqv-jmqgp.example.com` or `https://octops-g6qkw-gnp2h.example.com`

```yaml
# simplified Fleet manifest for Domain mode
# each GameServer is accessible using the combination: [gameserver_name].example.com
apiVersion: "agones.dev/v1"
kind: Fleet
metadata:
  name: fleet-us-east1-1
spec:
  replicas: 3
  template:
    metadata:
      annotations:
        octops.io/ingress-class-name: "contour" #required for Contour to handle ingress
        octops-projectcontour.io/websocket-routes: "/" #required for Contour to enable websocket
        octops.io/gameserver-ingress-mode: "domain"
        octops.io/gameserver-ingress-domain: "example.com"
```

Check the [examples](examples) folder for a full Fleet manifest that uses the `Domain` routing mode.

### Path
There is one global domain and gameservers are available using the URL path. I.e.: `https://servers.example.com/octops-2dnqv-jmqgp` or `https://servers.example.com/octops-g6qkw-gnp2h`

```yaml
# simplified Fleet manifest for Path mode
# each GameServer is accessible using the combination: servers.example.com/[gameserver_name]
apiVersion: "agones.dev/v1"
kind: Fleet
metadata:
  name: fleet-us-east1-1
spec:
  replicas: 3
  template:
    metadata:
      annotations:
        octops.io/ingress-class-name: "contour" #required for Contour to handle ingress
        octops-projectcontour.io/websocket-routes: "/{{ .Name }}" #required for Contour to enable websocket for exact path. This is a template that the controller will replace by the name of the game server
        octops.io/gameserver-ingress-mode: "path"
        octops.io/gameserver-ingress-fqdn: servers.example.com
```

Check the [examples](examples) folder for a full Fleet manifest that uses the `Path` routing mode.

## Kubernetes Gateway API (alternative to Ingress)

> **Experimental.** Gateway API support has been validated end-to-end but has not seen production usage. Please report bugs and feedback at https://github.com/Octops/gameserver-ingress-controller/issues.

The controller supports the [Kubernetes Gateway API](https://gateway-api.sigs.k8s.io/) as an alternative to the standard Ingress resource. Add the `octops.io/router-backend: gateway` annotation to your Fleet or GameServer to opt in. The default behaviour (Ingress) is unchanged.

### How it differs from Ingress

With Ingress the controller creates one `Ingress` resource per game server and cert-manager provisions one TLS certificate per game server. With Gateway API:

- The controller creates one `HTTPRoute` per game server — routing rules only, no TLS config.
- A shared `Gateway` resource (created once by ops) handles TLS termination using a single wildcard certificate (domain mode) or a single hostname certificate (path mode). This avoids Let's Encrypt rate limits entirely.
- cert-manager manages the certificate independently of individual game servers.

### Prerequisites

1. Gateway API CRDs and the Contour Gateway Provisioner installed — see the [Contour setup section](#requirements) above.
2. A `GatewayClass` named `contour` created pointing at the provisioner controller (also in the Contour setup section above).
3. cert-manager v1.15+ installed with a `ClusterIssuer` configured. Gateway API support is **Beta** as of cert-manager v1.15 (latest: v1.20.0) and must be enabled:
   ```yaml
   # helm values or cert-manager config
   config:
     enableGatewayAPI: true
   ```

### Setup (apply once, before deploying any Fleet)

```bash
# 1. Issue the TLS certificate (creates a wildcard cert via cert-manager)
kubectl apply -f examples/gateway/certificate.yaml

# 2. Create the Gateway (the provisioner will spin up a dedicated Envoy for it)
kubectl apply -f examples/gateway/gateway.yaml
```

Update the domain, issuer name, and `gatewayClassName` in both files to match your environment before applying.

### Deploy a Fleet

```yaml
# Domain mode — each game server gets its own subdomain: https://[gs-name].game.example.com
annotations:
  octops.io/router-backend: "gateway"
  octops.io/gateway-name: "gateway"
  octops.io/gateway-section-name: "https"
  octops.io/gameserver-ingress-mode: "domain"
  octops.io/gameserver-ingress-domain: "game.example.com"
```

```yaml
# Path mode — all game servers share one hostname: https://game.example.com/[gs-name]
annotations:
  octops.io/router-backend: "gateway"
  octops.io/gateway-name: "gateway"
  octops.io/gateway-section-name: "https"
  octops.io/gameserver-ingress-mode: "path"
  octops.io/gameserver-ingress-fqdn: "game.example.com"
```

Full manifests are available in [examples/gateway](examples/gateway):
- [`certificate.yaml`](examples/gateway/certificate.yaml) — cert-manager Certificate for TLS
- [`gateway.yaml`](examples/gateway/gateway.yaml) — shared Gateway resource
- [`fleet-domain.yaml`](examples/gateway/fleet-domain.yaml) — Fleet using domain routing mode
- [`fleet-path.yaml`](examples/gateway/fleet-path.yaml) — Fleet using path routing mode

### Gateway API annotations reference

| Annotation | Required | Description |
|---|---|---|
| `octops.io/router-backend` | Yes | Set to `gateway` to use Gateway API instead of Ingress |
| `octops.io/gateway-name` | Yes | Name of the pre-existing `Gateway` resource |
| `octops.io/gateway-namespace` | No | Namespace of the Gateway (defaults to same namespace as the game server) |
| `octops.io/gateway-section-name` | No | Listener name inside the Gateway (e.g. `https`) |
| `octops.io/gameserver-ingress-mode` | Yes | `domain` or `path` — same as Ingress mode |
| `octops.io/gameserver-ingress-domain` | domain mode | Base domain for game server subdomains |
| `octops.io/gameserver-ingress-fqdn` | path mode | Shared hostname for all game servers |

> **Note:** The annotations `octops.io/terminate-tls` and `octops.io/issuer-tls-name` have **no effect** in gateway mode. TLS is configured on the `Gateway` listener, not on individual `HTTPRoute` resources. The controller will emit a warning event if either annotation is found on a game server using the gateway backend.

### HTTPRoutes created by the controller

```bash
$ kubectl get httproutes
NAME                 HOSTNAMES                               AGE
octops-2dnqv-jmqgp   ["octops-2dnqv-jmqgp.game.example.com"]   4m
octops-2dnqv-d9nxd   ["octops-2dnqv-d9nxd.game.example.com"]   4m
octops-2dnqv-fr8tx   ["octops-2dnqv-fr8tx.game.example.com"]   4m
```

## How it works
When a game server is created by Agones, either as part of a Fleet or a stand-alone deployment, the Octops controller will handle the provisioning of a couple of resources.

It will use the information present in the game server annotations and metadata to create the required Ingress and dependencies.

Below is an example of a manifest that deploys a Fleet using the `Domain` routing mode:
```yaml
# Reference: https://agones.dev/site/docs/reference/fleet/
apiVersion: "agones.dev/v1"
kind: Fleet
metadata:
  name: octops # the name of your fleet
  labels: # optional labels
    cluster: gke-1.24
    region: us-east-1
spec:
  replicas: 3
  template:
    metadata:
      labels: # optional labels
        cluster: gke-1.24
        region: us-east-1
      annotations:
        octops.io/ingress-class-name: "contour" # required for Contour to handle ingress
        octops-projectcontour.io/websocket-routes: "/" # required for Contour to enable websocket
        # Required annotation used by the controller
        octops.io/gameserver-ingress-mode: "domain"
        octops.io/gameserver-ingress-domain: "example.com"
        octops.io/terminate-tls: "true"
        octops.io/issuer-tls-name: "letsencrypt-prod"
# The rest of your fleet spec stays the same        
 ...
```

Deployed GameServers:
```bash
# kubectl [-n yournamespace] get gs
NAME                 STATE   ADDRESS         PORT   NODE     AGE
octops-2dnqv-jmqgp   Ready   36.23.134.23    7437   node-1   10m
octops-2dnqv-d9nxd   Ready   36.23.134.23    7323   node-1   10m
octops-2dnqv-fr8tx   Ready   32.76.142.33    7779   node-2   10m
```

Ingresses created by the controller:
```bash
# kubectl [-n yournamespace] get ingress
NAME                 HOSTS                           ADDRESS         PORTS     AGE
octops-2dnqv-jmqgp   octops-2dnqv-jmqgp.example.com                   80, 443   4m48s
octops-2dnqv-d9nxd   octops-2dnqv-d9nxd.example.com                   80, 443   4m46s
octops-2dnqv-fr8tx   octops-2dnqv-fr8tx.example.com                   80, 443   4m45s
```

Proxy Mapping - Ingress x GameServer 
```bash
# The game server public domain uses the omitted 443/HTTPS port instead of the Agones port range 7000-8000
https://octops-2dnqv-jmqgp.example.com/ ⇢ octops-2dnqv-jmqgp:7437
https://octops-2dnqv-d9nxd.example.com/ ⇢ octops-2dnqv-d9nxd:7323
https://octops-2dnqv-fr8tx.example.com/ ⇢ octops-2dnqv-fr8tx:7779
```

## Conventions
The table below shows how the information from the game server is used to compose the ingress settings.

| Game Server                                     |           Ingress           | 
|-------------------------------------------------|:---------------------------:|
| name                                            |      [hostname, path]       | 
| annotation: octops.io/gameserver-ingress-mode   |       [domain, path]        |
| annotation: octops.io/gameserver-ingress-domain |         base domain         |
| annotation: octops.io/gameserver-ingress-fqdn   |        global domain        | 
| annotation: octops.io/terminate-tls             | terminate TLS (true, false) |
| annotation: octops.io/issuer-tls-name           |  name of the ClusterIssuer  |
| annotation: octops-[custom-annotation]          |      custom-annotation      |
| annotation: octops.io/tls-secret-name           |    custom ingress secret    |
| annotation: octops.io/ingress-class-name        |   ingressClassName field    |

**Support for Multiple Domains**

For both routing modes one can specify multiple domains. That will make the same game server to be accessible through all of them.

The value must be a comma separated list of domains.

```yaml
annotations:
  # Domain Mode
  octops.io/gameserver-ingress-domain: "example.com,example.gg"
  # Path Mode
  octops.io/gameserver-ingress-fqdn: "www.example.com,www.example.gg"
```

### Custom Annotations
Any Fleet or GameServer resource annotation that contains the prefix `octops-` will be added down to the Ingress resource crated by the Octops controller.

`octops-projectcontour.io/websocket-routes`: `/`

Will be added to the ingress in the following format:

`projectcontour.io/websocket-routes`: `/`

The same way annotations prefixed with `octops.service-` will be passed down to the service resource that is the bridge between the game server and the ingress.

`octops.service-myannotation`: `myvalue`

Will be added to the service in the following format:

`myannotation`: `myvalue`

### Templates
It is also possible to use a template to fill values at the Ingress and Services creation time. 

This feature is specially useful if the routing mode is `path`. Envoy will only enable websocket for routes that match exactly the path set on the Ingress rules.

The example below demonstrates how custom annotations using a template would be generated for a game server named `octops-tl6hf-fnmgd`.

```yaml
# manifest.yaml
octops-projectcontour.io/websocket-routes: "/{{ .Name }}"

# parsed
octops-projectcontour.io/websocket-routes: "/octops-tl6hf-fnmgd"
```


The field `.Port` is the port exposed by the game server that was assigned by Agones.

```yaml
# manifest.yaml
octops.service-projectcontour.io/upstream-protocol.tls: "{{ .Port }}"

# parsed
octops.service-projectcontour.io/upstream-protocol.tls: "7708"
```

**Important**

If you are deploying manifests using helm you should scape special characters.

```yaml
# manifest.yaml
octops.service-projectcontour.io/upstream-protocol.tls: '{{"{{"}} .Port {{"}}"}}'

# parsed
octops.service-projectcontour.io/upstream-protocol.tls: "7708"
```

The same applies for any other custom annotation. The currently supported GameServer fields are `.Name` and `.Port`. More to be added in the future.

**Any annotation can be used. It is not restricted to the [Contour controller annotations](https://projectcontour.io/docs/main/config/annotations/)**.

`octops-my-custom-annotations`: `my-custom-value` will be passed to the Ingress resource as:

`my-custom-annotations`: `my-custom-value`

Multiline is also supported, I.e.:

```yaml
annotations:
    octops-example.com/backend-config-snippet: |
      http-send-name-header x-dst-server
      stick-table type string len 32 size 100k expire 30m
      stick on req.cook(sessionid)
```

**Remember that the max length of a label name is 63 characters. That limit is imposed by Kubernetes**

https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/
 
## Fleet and GameServer Resource Manifests

- **octops.io/gameserver-ingress-mode:** defines the ingress routing mode, possible values are: domain or path.
- **octops.io/gameserver-ingress-domain:** name of the domain to be used when creating the ingress. This is the public domain that players will use to reach out to the dedicated game server.
- **octops.io/gameserver-ingress-fqdn:** full domain name where gameservers will be accessed based on the URL path.
- **octops.io/terminate-tls:** it determines if the ingress will terminate TLS. If set to "false" it means that TLS will be terminated at the load balancer. In this case there won't be a certificate issued by the local cert-manager.
- **octops.io/issuer-tls-name:** required if `terminate-tls=true` and certificates are provisioned by CertManager. This is the name of the ClusterIssuer that cert-manager will use when creating the certificate for the ingress.
- **octops.io/ingress-class-name:** Defines the ingress class name to be used e.g ("contour", "nginx", "traefik")

The same configuration works for Fleets and GameServers. Add the following annotations to your manifest:
```yaml
# Fleet annotations using ingress routing mode: domain
annotations:
  octops.io/ingress-class-name: "contour" # required for Contour to handle ingress
  octops-projectcontour.io/websocket-routes: "/" # required for Contour to enable websocket
  octops.io/gameserver-ingress-mode: "domain"
  octops.io/gameserver-ingress-domain: "example.com"
  octops.io/terminate-tls: "true"
  octops.io/issuer-tls-name: "selfsigned-issuer"
```

```yaml
# Fleet annotations using ingress routing mode: path
annotations:
  octops.io/ingress-class-name: "contour" # required for Contour to handle ingress
  octops-projectcontour.io/websocket-routes: "/" # required for Contour to enable websocket
  octops.io/gameserver-ingress-mode: "path"
  octops.io/gameserver-ingress-fqdn: "servers.example.com"
  octops.io/terminate-tls: "true"
  octops.io/issuer-tls-name: "selfsigned-issuer"
```

```yaml
# Optional and can be ignored if TLS is not terminated by the ingress controller
octops.io/terminate-tls: "true"
octops.io/issuer-tls-name: "selfsigned-issuer"
```

# Wildcard Certificates
It is worth noticing that games using the domain routing model and CertManager handling certificates, might face a limitation imposed by Letsencrypt in terms of the numbers of certificates that can be issued per week. One can find information about the rate limiting on https://letsencrypt.org/docs/rate-limits/.

For each new game server created there will be a new certificate request triggered by CertManager. That means that `https://octops-2dnqv-jmqgp.example.com` and `https://octops-2dnqv-d9nxd.example.com` require 2 different certificates. That approach will not scale well for games that have a high churn. In fact Letsencrypt limits to 50 domains per week.

In order to avoid issues with certificates and limits one should implement a wildcard certificate. There are different ways that this can be achieved. It also depends on how your cloud provider handled TLS termination at the load balancer or how the DNS and certificates for the game domain are managed.

There are 2 options:
1. Terminate TLS at the load balancer that is exposed by the Contour/Envoy service. That way one can ignore all the TLS or issuer annotations. That also removes the dependency on CertManager. Be aware that cloud providers have different implementations of how certificates are generated and managed. Moreover, how they are assigned to public endpoints or load balancers.
2. Provide a self-managed wildcard certificate.  
   1. Add a [TLS secret](https://kubernetes.io/docs/concepts/configuration/secret/#tls-secrets) to the `default` namespace that holds the wildcard certificate content. That certificate must have been generated, acquired or bought from a different source.
   2. Set the annotation `octops.io/terminate-tls: "true"`. That will instruct the controller to add the TLS section to the Ingress.
   3. Add the annotation `octops.io/tls-secret-name: "my-wildcard-cert"`. That secret will be added to the Ingress under the TLS section. It will tell Envoy to use that secret content to terminate TLS for the public game server endpoint.

**Important**
- Certificate renewal should be handled by the game server owner. The fact that the secret exists does not mean that Kubernetes or any other process will handle expiration.
- CertManager can be used to generate wildcard certificates using [DNS validation](https://cert-manager.io/docs/tutorials/acme/dns-validation/#issuing-an-acme-certificate-using-dns-validation).  

# Clean up and GameServer Lifecycle
Every resource created by the Octops controller is attached to the game server itself. That means, when a game server is deleted from the cluster all its dependencies will be cleaned up by the Kubernetes garbage collector.

**Manual deletion of services and ingresses is not required by the operator of the cluster.**

# How to install the Octops Controller

Deploy the controller running:
```bash
$ kubectl apply -f deploy/install.yaml
or
$ kubectl apply -f https://raw.githubusercontent.com/Octops/gameserver-ingress-controller/main/deploy/install.yaml
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

## Controller Flags

| Flag | Default | Description |
|---|---|---|
| `--kubeconfig` | `` | Path to kubeconfig file. Not required when running in-cluster. |
| `--sync-period` | `15s` | Minimum frequency at which watched resources are reconciled. |
| `--webhook-port` | `30234` | Port used for webhooks. |
| `--health-probe-addrs` | `:30235` | Address for liveness/readiness probes (`/healthz`). |
| `--metrics-addrs` | `:9090` | Address for Prometheus metrics. |
| `--max-concurrent-reconciles` | `10` | Maximum number of concurrent reconcile loops. |
| `--verbose` | `false` | Enable verbose logging. |
| `--enable-gateway-api` | `auto` | Controls the Gateway API backend — see below. |

### `--enable-gateway-api`

This flag controls whether the controller creates a Gateway API (`HTTPRoute`) informer at startup. It accepts three values:

| Value | Behaviour |
|---|---|
| `auto` (default) | Probe the cluster at startup. If `httproutes.gateway.networking.k8s.io` CRD is present, enable the Gateway API backend. If absent, log a warning and disable it — the controller starts normally and the Ingress backend continues to work. |
| `true` | Always enable. Fail hard at startup if the HTTPRoute CRD is not installed. Use this to make a missing CRD a deployment error rather than a silent degradation. |
| `false` | Always disable. No informer or client for Gateway API is created. Use this to keep the controller lightweight in clusters that will never use the gateway backend. |

The default `auto` mode is safe for clusters that have not installed Gateway API CRDs — the controller will start and continue to manage Ingress resources normally. If a game server uses `octops.io/router-backend: gateway` while the backend is disabled, the controller will log a clear error for that specific game server rather than silently falling back to Ingress.

## Events
You can track events recorded for each GameServer running `kubectl get events [-w]` and the output will look similar to:
```
...
1s Normal  Creating  gameserver/octops-domain-tqmvm-rcl5p  Creating Service for gameserver default/octops-domain-tqmvm-rcl5p
0s Normal  Created   gameserver/octops-domain-tqmvm-rcl5p  Service created for gameserver default/octops-domain-tqmvm-rcl5p
0s Normal  Creating  gameserver/octops-domain-tqmvm-rcl5p  Creating Ingress for gameserver default/octops-domain-tqmvm-rcl5p
0s Normal  Created   gameserver/octops-domain-tqmvm-rcl5p  Ingress created for gameserver default/octops-domain-tqmvm-rcl5p
...
```

The controller will record errors if a resource can't be created.
```
0s Warning Failed  gameserver/octops-domain-zxt2q-6xl6r  Failed to create Ingress for gameserver default/octops-domain-zxt2q-6xl6r: ingress routing mode domain requires the annotation octops.io/gameserver-ingress-domain to be present on octops-domain-zxt2q-6xl6r, check your Fleet or GameServer manifest.
```

Alternatively, you can check events for a particular game server running
```
$ kubectl describe gameserver [gameserver-name]
...
Events:
  Type    Reason          Age    From                           Message
  ----    ------          ----   ----                           -------
  Normal  PortAllocation  2m59s  gameserver-controller          Port allocated
  Normal  Creating        2m59s  gameserver-controller          Pod octops-domain-4sk5v-7gtw4 created
  Normal  Scheduled       2m59s  gameserver-controller          Address and port populated
  Normal  RequestReady    2m53s  gameserver-sidecar             SDK state change
  Normal  Ready           2m53s  gameserver-controller          SDK.Ready() complete
  Normal  Creating        2m53s  gameserver-ingress-controller  Creating Service for gameserver default/octops-domain-4sk5v-7gtw4
  Normal  Created         2m53s  gameserver-ingress-controller  Service created for gameserver default/octops-domain-4sk5v-7gtw4
  Normal  Creating        2m53s  gameserver-ingress-controller  Creating Ingress for gameserver default/octops-domain-4sk5v-7gtw4
  Normal  Created         2m53s  gameserver-ingress-controller  Ingress created for gameserver default/octops-domain-4sk5v-7gtw4
```

## Extras

Infrastructure manifests are organised by backend:

- **Ingress mode**: ClusterIssuer examples are in [deploy/cert-manager](deploy/cert-manager). Update the email and ingress class before applying.
- **Gateway API mode**: The `GatewayClass` resource is in [deploy/gateway](deploy/gateway). Apply it after installing the Contour Gateway Provisioner.

For a quick test you can use the [examples/fleet.yaml](examples/fleet.yaml). This manifest will deploy a simple http game server that keeps the health check and changes the state to "Ready".
```bash
$ kubectl apply -f examples/fleet-domain.yaml

# Find the ingress for one of the replicas
$ kubectl get ingress
NAME                 HOSTS                           ADDRESS         PORTS     AGE
octops-tl6hf-fnmgd   octops-tl6hf-fnmgd.example.com                   80, 443   67m
octops-tl6hf-jjqvt   octops-tl6hf-jjqvt.example.com                   80, 443   67m
octops-tl6hf-qzhzb   octops-tl6hf-qzhzb.example.com                   80, 443   67m

# Test the public endpoint. You will need a valid public domain or some network sorcery depending on the environment you pushed the manifest.
$ curl https://octops-tl6hf-fnmgd.example.com

# Output
{"Name":"octops-tl6hf-fnmgd","Address":"36.23.134.23:7318","Status":{"state":"Ready","address":"192.168.0.117","ports":[{"name":"default","port":7318}]}}
```

### Ingress manifest
Below is an example of a manifest created by the controller for a GameServer from a Fleet set to routing mode `domain`:
```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  labels:
    agones.dev/gameserver: "octops-tl6hf-fnmgd"
    kubernetes.io/ingress.class: "contour"
    projectcontour.io/websocket-routes: "/"
  name: octops-tl6hf-fnmgd
  namespace: default
spec:
  rules:
    - host: octops-tl6hf-fnmgd.example.com
      http:
        paths:
          - backend:
              service:
                name: octops-tl6hf-fnmgd #service is also created but the controller
                port:
                  number: 7837
            path: /
            pathType: Prefix
  tls:
    - hosts:
        - octops-tl6hf-fnmgd.example.com
      secretName: octops-tl6hf-fnmgd-tls
```

# Demo

To demonstrate how the Octops controller workers, you can deploy a fleet of Quake 3 servers (QuakeKube) that can be managed by Agones.

> QuakeKube is a Kubernetes-fied version of QuakeJS that runs a dedicated Quake 3 server in a Kubernetes Deployment, and then allow clients to connect via QuakeJS in the browser.

The source code of the project that integrates the game with Agones can be found on https://github.com/Octops/quake-kube. 

It is a fork from the original project https://github.com/criticalstack/quake-kube.

## Deploy the Quake Fleet

Update the fleet annotation and use a domain that you can point your load balancer or Public IP.
```yaml
# examples/quake/quake-fleet.yaml
annotations:
  octops.io/gameserver-ingress-domain: "yourdomain.com" # Do not include the host part. The host name will be generated by the controller and it is individual to each gameserver.
```

Deploy the manifest
```bash
$ kubectl apply -f examples/quake/quake-fleet.yaml
```

When the game becomes `Ready` the `gameserver-ingress-controller` will create the Ingress that holds the public URL. Use the following command to list all the ingresses.

```bash
$ kubectl get ingress

# Output
NAME                 HOSTS                                ADDRESS         PORTS     AGE
octops-w2lpj-wtqwm   octops-w2lpj-wtqwm.yourdomain.com                    80, 443   18m
```

Point your browser to the address from the `HOST` column. Depending on your setup there may be a warning about certificates.

Destroy

You can destroy the quake fleet running:
```bash
$ kubectl destroy -f examples/quake/quake-fleet.yaml 
```

As expected Agones will destroy the Fleet, consequently deleting all the Ingresses associated to the destroyed gameservers. 


## Screenshots

**The screenshots below use a fake domain `arena.com` used just for local and demonstration purpose. That domain should reflect the name of the domain you own and want your gameservers to be hosted. On real cloud environment, the certificate issued by cert-manager will be valid.**

![alt text](docs/screenshots/quake2.png)

![alt text](docs/screenshots/quake1.png)

![alt text](docs/screenshots/quake3.png)
