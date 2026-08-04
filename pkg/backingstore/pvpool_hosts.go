package backingstore

import (
	"fmt"
	"strings"

	"github.com/noobaa/noobaa-operator/v5/pkg/nb"
	"github.com/noobaa/noobaa-operator/v5/pkg/options"

	corev1 "k8s.io/api/core/v1"
)

// hostPodName returns the kubernetes pod name encoded in a noobaa host name.
// Host names from list_hosts are formatted as "<pod-name>#<host-sequence>".
func hostPodName(hostName string) string {
	if i := strings.Index(hostName, "#"); i >= 0 {
		return hostName[:i]
	}
	return hostName
}

// pvPoolAgentPostfix extracts the random postfix from a PV-pool agent pod name.
// Pod names are formatted as "<backingstore>-<system>-pod-<postfix>".
func pvPoolAgentPostfix(podName, backingStoreName string) string {
	prefix := fmt.Sprintf("%s-%s-pod-", backingStoreName, options.SystemName)
	if !strings.HasPrefix(podName, prefix) {
		return ""
	}
	return podName[len(prefix):]
}

// pvPoolPVCName returns the expected PVC name for a PV-pool agent postfix.
func pvPoolPVCName(backingStoreName, postfix string) string {
	return fmt.Sprintf("%s-%s-pvc-%s", backingStoreName, options.SystemName, postfix)
}

// findPodByName returns true if a pod with the given name exists.
func findPodByName(pods []corev1.Pod, name string) bool {
	for i := range pods {
		if pods[i].Name == name {
			return true
		}
	}
	return false
}

// findPVCByName returns true if a PVC with the given name exists.
func findPVCByName(pvcs []corev1.PersistentVolumeClaim, name string) bool {
	for i := range pvcs {
		if pvcs[i].Name == name {
			return true
		}
	}
	return false
}

// isOrphanedPvPoolHost reports whether the host has no matching agent pod and no
// matching PVC. A missing pod with an existing PVC is not treated as orphaned
// because the operator may be recreating the pod (e.g. image/env update).
func isOrphanedPvPoolHost(host nb.HostInfo, pods []corev1.Pod, pvcs []corev1.PersistentVolumeClaim, backingStoreName string) bool {
	if host.Mode == "DELETING" {
		return false
	}
	podName := hostPodName(host.Name)
	if findPodByName(pods, podName) {
		return false
	}
	postfix := pvPoolAgentPostfix(podName, backingStoreName)
	if postfix == "" {
		// Unexpected host naming; treat as orphaned so inventory can heal.
		return true
	}
	return !findPVCByName(pvcs, pvPoolPVCName(backingStoreName, postfix))
}

// findOrphanedPvPoolHosts returns hosts that no longer have a matching pod/PVC.
func findOrphanedPvPoolHosts(hosts []nb.HostInfo, pods []corev1.Pod, pvcs []corev1.PersistentVolumeClaim, backingStoreName string) []nb.HostInfo {
	orphans := make([]nb.HostInfo, 0)
	for i := range hosts {
		if isOrphanedPvPoolHost(hosts[i], pods, pvcs, backingStoreName) {
			orphans = append(orphans, hosts[i])
		}
	}
	return orphans
}

// countAttachedPvPoolHosts counts hosts that still map to a running/existing agent pod.
// Hosts in DELETING mode and orphaned hosts (no matching pod) are excluded so that
// stale DB entries do not look like an unsupported scale-down.
func countAttachedPvPoolHosts(hosts []nb.HostInfo, pods []corev1.Pod) int {
	count := 0
	for i := range hosts {
		if hosts[i].Mode == "DELETING" {
			continue
		}
		if findPodByName(pods, hostPodName(hosts[i].Name)) {
			count++
		}
	}
	return count
}
