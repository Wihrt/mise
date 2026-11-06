#!/usr/bin/env python3
"""List the rules of every CiliumNetworkPolicy in a rendered Helm YAML file.

Usage: python3 -I policy-rules.py render.yaml   (needs pyyaml)
"""
import sys

import yaml


def peers(rule):
    out = []
    for key in ("fromEndpoints", "toEndpoints"):
        for selector in rule.get(key) or []:
            labels = selector.get("matchLabels") or {}
            text = ",".join(f"{k.split('/')[-1]}={v}" for k, v in labels.items())
            out.append(text or "all pods")
    for key in ("fromEntities", "toEntities"):
        out += ["entity:" + str(e) for e in rule.get(key) or []]
    for key in ("fromCIDR", "toCIDR"):
        out += rule.get(key) or []
    for fqdn in rule.get("toFQDNs") or []:
        out.append("fqdn:" + str(fqdn.get("matchName") or fqdn.get("matchPattern")))
    return out


def main(path):
    with open(path) as fh:
        docs = [d for d in yaml.safe_load_all(fh) if d]
    policies = [d for d in docs if d.get("kind") == "CiliumNetworkPolicy"]
    if not policies:
        print("no CiliumNetworkPolicy rendered")
    for policy in policies:
        name = policy["metadata"]["name"]
        for direction in ("ingress", "egress"):
            for rule in (policy.get("spec") or {}).get(direction) or []:
                ports = [
                    f"{p.get('port')}/{p.get('protocol', 'ANY')}"
                    for block in rule.get("toPorts") or []
                    for p in block.get("ports") or []
                ]
                print(f"{name} {direction} ports={ports or 'all'} peers={peers(rule) or ['none']}")


if __name__ == "__main__":
    if len(sys.argv) != 2:
        sys.exit("usage: policy-rules.py FILE")
    main(sys.argv[1])
