package main

# PSS restricted fields that kubeconform (schema only) cannot check.
# Input: rendered manifests (kustomize build / helm template output).

workloads := {"Deployment", "StatefulSet", "DaemonSet", "Job", "ReplicaSet"}

pod_spec := input.spec.template.spec if input.kind in workloads
pod_spec := input.spec if input.kind == "Pod"
pod_spec := input.spec.jobTemplate.spec.template.spec if input.kind == "CronJob"

containers contains c if some c in pod_spec.containers
containers contains c if some c in pod_spec.initContainers

deny contains msg if {
	some c in containers
	object.get(c, ["securityContext", "allowPrivilegeEscalation"], null) != false
	msg := sprintf("%s/%s container %s: allowPrivilegeEscalation must be false", [input.kind, input.metadata.name, c.name])
}

deny contains msg if {
	some c in containers
	object.get(c, ["securityContext", "readOnlyRootFilesystem"], null) != true
	msg := sprintf("%s/%s container %s: readOnlyRootFilesystem must be true", [input.kind, input.metadata.name, c.name])
}

deny contains msg if {
	some c in containers
	not "ALL" in object.get(c, ["securityContext", "capabilities", "drop"], [])
	msg := sprintf("%s/%s container %s: capabilities.drop must contain ALL", [input.kind, input.metadata.name, c.name])
}

deny contains msg if {
	some c in containers
	caps := object.get(c, ["securityContext", "capabilities", "add"], [])
	some cap in caps
	cap != "NET_BIND_SERVICE"
	msg := sprintf("%s/%s container %s: capability %s not allowed (only NET_BIND_SERVICE)", [input.kind, input.metadata.name, c.name, cap])
}

deny contains msg if {
	some c in containers
	object.get(c, ["securityContext", "privileged"], false) == true
	msg := sprintf("%s/%s container %s: privileged must not be set", [input.kind, input.metadata.name, c.name])
}

# runAsNonRoot and seccomp may be set on the pod or on the container.
deny contains msg if {
	some c in containers
	not run_as_non_root(c)
	msg := sprintf("%s/%s container %s: runAsNonRoot must be true (pod or container)", [input.kind, input.metadata.name, c.name])
}

deny contains msg if {
	some c in containers
	not seccomp_ok(c)
	msg := sprintf("%s/%s container %s: seccompProfile.type must be RuntimeDefault or Localhost", [input.kind, input.metadata.name, c.name])
}

deny contains msg if {
	some c in containers
	not numeric_uid(c)
	msg := sprintf("%s/%s container %s: runAsUser must be a non-zero numeric uid (pod or container)", [input.kind, input.metadata.name, c.name])
}

deny contains msg if {
	some k in ["hostNetwork", "hostPID", "hostIPC"]
	object.get(pod_spec, k, false) == true
	msg := sprintf("%s/%s: %s must not be true", [input.kind, input.metadata.name, k])
}

deny contains msg if {
	some v in object.get(pod_spec, "volumes", [])
	"hostPath" in object.keys(v)
	msg := sprintf("%s/%s: hostPath volume %s not allowed", [input.kind, input.metadata.name, v.name])
}

deny contains msg if {
	some c in containers
	some p in object.get(c, "ports", [])
	"hostPort" in object.keys(p)
	msg := sprintf("%s/%s container %s: hostPort must not be set", [input.kind, input.metadata.name, c.name])
}

run_as_non_root(c) if object.get(c, ["securityContext", "runAsNonRoot"], null) == true
run_as_non_root(c) if {
	object.get(c, ["securityContext", "runAsNonRoot"], null) == null
	object.get(pod_spec, ["securityContext", "runAsNonRoot"], null) == true
}

seccomp_ok(c) if object.get(c, ["securityContext", "seccompProfile", "type"], "") in {"RuntimeDefault", "Localhost"}
seccomp_ok(c) if {
	object.get(c, ["securityContext", "seccompProfile"], null) == null
	object.get(pod_spec, ["securityContext", "seccompProfile", "type"], "") in {"RuntimeDefault", "Localhost"}
}

numeric_uid(c) if nonzero_uid(object.get(c, ["securityContext", "runAsUser"], null))
numeric_uid(c) if {
	object.get(c, ["securityContext", "runAsUser"], null) == null
	nonzero_uid(object.get(pod_spec, ["securityContext", "runAsUser"], null))
}

nonzero_uid(u) if {
	is_number(u)
	u != 0
}
