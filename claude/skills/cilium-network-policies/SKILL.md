---
name: cilium-network-policies
description: Use when writing or reviewing a CiliumNetworkPolicy or CiliumClusterwideNetworkPolicy, moving a namespace to default-deny, or diagnosing a connection that times out or is refused on a cluster running Cilium.
category: kubernetes
---

Policies are allow lists by identity. Write them from the labels the pods really carry and the flows really observed, and prove each one by a flow in Hubble.

## Model
- A pod is default-deny in one direction (ingress or egress) as soon as one policy selects it with a rule for that direction; before that, the direction is open. Rules add up; a deny rule wins over any allow.
- The first egress policy on a pod cuts its DNS. Every egress policy goes with an allow to the cluster DNS on port 53, UDP and TCP.
- `CiliumNetworkPolicy` selects pods of its own namespace, and a peer selector without a namespace label also means that namespace. To name another namespace, add `k8s:io.kubernetes.pod.namespace: <ns>` to the peer. `CiliumClusterwideNetworkPolicy` has no namespace: every selector is cluster-wide.
- Peers by kind: pods with `fromEndpoints` / `toEndpoints`; the node, the API server and the outside with entities (`host`, `remote-node`, `kube-apiserver`, `world`, `cluster`); outside addresses with `toCIDR`, which never matches a pod or a node of the cluster.
- Ports are the pod's, not the Service's: the policy is enforced after the Service is resolved, so write the `targetPort` (the container port).

## Writing
- Take selectors from the live pods or the render, never from the name of a Deployment: `kubectl get pods -n <ns> --show-labels`. A selector that matches nothing is accepted and does nothing: the pods stay open, or stay in default-deny from another policy.
- Rules shared by many workloads (DNS, API server, metrics scraping, the ingress controller) are clusterwide policies keyed on a label the workload opts into, not copies in each namespace. An application's own file only holds its own flows.
- `toFQDNs` works only when the pod's DNS queries go through Cilium's DNS proxy: the egress rule to the cluster DNS carries `rules.dns` with a `matchPattern`. `*.example.com` does not match `example.com`: list both.
- An L7 rule (`rules.http`) sends the traffic through the proxy and turns a drop into a 403. Use it only when the criterion is about paths or methods.
- Policies are delivered by Git like any manifest (`gitops-flux-argocd`); never `kubectl apply` one to try it on a shared cluster.

## Diagnosis
- A timeout is a drop: a policy. `Connection refused` reached the destination and nothing listens there, or a proxy answered: not a policy.
- Read the drop before changing anything: `hubble observe --namespace <ns> --verdict DROPPED --last 50` (or `--pod <ns>/<pod>`). The line gives source identity, destination, port and direction; write the rule for that flow only.
- Enforcement per pod: `kubectl get ciliumendpoints -n <ns>` shows whether ingress and egress are enforced. A pod expected under policy and shown as not enforced means no selector matched it.
- DNS resolves and the connection still times out: the DNS allow is there and the destination rule is missing or its selector is silent.

## Moving a namespace to default-deny
- Observe first: collect the namespace's flows in Hubble over a period that covers its batch jobs and backups. Write the allows from them, then enforce, one namespace at a time.
- Do not forget what talks to the pods from outside the application: kubelet probes (from the node), the ingress controller, metrics scraping, the backup agent, the operator's webhooks.
- Proof of a policy: one observed flow `FORWARDED` for what it allows, one `DROPPED` for what it must block. Quote both lines.
