package backingstore

import (
	"testing"

	"github.com/noobaa/noobaa-operator/v5/pkg/nb"
	"github.com/noobaa/noobaa-operator/v5/pkg/options"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestHostPodName(t *testing.T) {
	tests := []struct {
		name string
		host string
		want string
	}{
		{
			name: "with sequence",
			host: "noobaa-default-backing-store-noobaa-pod-210c66bb#3",
			want: "noobaa-default-backing-store-noobaa-pod-210c66bb",
		},
		{
			name: "without sequence",
			host: "noobaa-default-backing-store-noobaa-pod-210c66bb",
			want: "noobaa-default-backing-store-noobaa-pod-210c66bb",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hostPodName(tt.host); got != tt.want {
				t.Fatalf("hostPodName(%q) = %q, want %q", tt.host, got, tt.want)
			}
		})
	}
}

func TestPvPoolAgentPostfix(t *testing.T) {
	bs := "noobaa-default-backing-store"
	pod := bs + "-" + options.SystemName + "-pod-210c66bb"
	if got := pvPoolAgentPostfix(pod, bs); got != "210c66bb" {
		t.Fatalf("pvPoolAgentPostfix() = %q, want %q", got, "210c66bb")
	}
	if got := pvPoolAgentPostfix("unrelated-pod", bs); got != "" {
		t.Fatalf("pvPoolAgentPostfix(unrelated) = %q, want empty", got)
	}
}

func TestIsOrphanedPvPoolHost(t *testing.T) {
	bs := "noobaa-default-backing-store"
	podName := bs + "-" + options.SystemName + "-pod-210c66bb"
	pvcName := bs + "-" + options.SystemName + "-pvc-210c66bb"
	host := nb.HostInfo{Name: podName + "#1"}

	pods := []corev1.Pod{{ObjectMeta: metav1.ObjectMeta{Name: podName}}}
	pvcs := []corev1.PersistentVolumeClaim{{ObjectMeta: metav1.ObjectMeta{Name: pvcName}}}

	if isOrphanedPvPoolHost(host, pods, pvcs, bs) {
		t.Fatal("host with matching pod should not be orphaned")
	}
	if isOrphanedPvPoolHost(host, nil, pvcs, bs) {
		t.Fatal("host with matching PVC but no pod should not be orphaned (pod may be recreating)")
	}
	if !isOrphanedPvPoolHost(host, nil, nil, bs) {
		t.Fatal("host with no pod and no PVC should be orphaned")
	}

	deleting := nb.HostInfo{Name: podName + "#1", Mode: "DELETING"}
	if isOrphanedPvPoolHost(deleting, nil, nil, bs) {
		t.Fatal("DELETING hosts should not be treated as orphaned")
	}
}

func TestCountAttachedPvPoolHosts(t *testing.T) {
	bs := "noobaa-default-backing-store"
	podA := bs + "-" + options.SystemName + "-pod-aaaa"
	podB := bs + "-" + options.SystemName + "-pod-bbbb"
	podStale := bs + "-" + options.SystemName + "-pod-stale"

	hosts := []nb.HostInfo{
		{Name: podA + "#1"},
		{Name: podB + "#2"},
		{Name: podStale + "#3"},                 // orphaned - no pod
		{Name: podB + "#4", Mode: "DELETING"}, // ignored even though pod exists
	}
	pods := []corev1.Pod{
		{ObjectMeta: metav1.ObjectMeta{Name: podA}},
		{ObjectMeta: metav1.ObjectMeta{Name: podB}},
	}

	got := countAttachedPvPoolHosts(hosts, pods)
	if got != 2 {
		t.Fatalf("countAttachedPvPoolHosts() = %d, want 2", got)
	}

	// Stale host must not make attached count look like scale-down when numVolumes == 2.
	if got > 2 {
		t.Fatalf("orphaned hosts were counted toward attached hosts")
	}
}

func TestFindOrphanedPvPoolHosts(t *testing.T) {
	bs := "noobaa-default-backing-store"
	livePod := bs + "-" + options.SystemName + "-pod-live"
	stalePod := bs + "-" + options.SystemName + "-pod-stale"

	hosts := []nb.HostInfo{
		{Name: livePod + "#1"},
		{Name: stalePod + "#2"},
	}
	pods := []corev1.Pod{{ObjectMeta: metav1.ObjectMeta{Name: livePod}}}
	pvcs := []corev1.PersistentVolumeClaim{{
		ObjectMeta: metav1.ObjectMeta{Name: bs + "-" + options.SystemName + "-pvc-live"},
	}}

	orphans := findOrphanedPvPoolHosts(hosts, pods, pvcs, bs)
	if len(orphans) != 1 {
		t.Fatalf("findOrphanedPvPoolHosts() returned %d orphans, want 1", len(orphans))
	}
	if hostPodName(orphans[0].Name) != stalePod {
		t.Fatalf("unexpected orphan %q", orphans[0].Name)
	}
}
