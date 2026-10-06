package relationships

import (
	"context"

	"github.com/kubilitics/kubilitics-backend/internal/topology/v2"
)

// EventMatcher produces edges from Events to their involvedObject (e.g., Event→Pod, Event→Node).
type EventMatcher struct{}

func (EventMatcher) Name() string { return "event" }

func (m *EventMatcher) Match(ctx context.Context, bundle *v2.ResourceBundle) ([]v2.TopologyEdge, error) {
	if bundle == nil {
		return nil, nil
	}
	var edges []v2.TopologyEdge
	seen := make(map[string]bool)
	index := buildResourceExistenceIndex(bundle)

	for i := range bundle.Events {
		ev := &bundle.Events[i]
		kind := ev.InvolvedObject.Kind
		ns := ev.InvolvedObject.Namespace
		name := ev.InvolvedObject.Name
		if kind == "" || name == "" {
			continue
		}
		if !index.has(kind, ns, name) {
			continue
		}
		src := v2.NodeID("Event", ev.Namespace, ev.Name)
		tgt := v2.NodeID(kind, ns, name)
		id := v2.EdgeID(src, tgt, "event")
		if seen[id] {
			continue
		}
		seen[id] = true
		edges = append(edges, v2.TopologyEdge{
			ID:                   id,
			Source:               src,
			Target:               tgt,
			RelationshipType:     "event",
			RelationshipCategory: "cluster",
			Label:                ev.Reason,
			Detail:               "involvedObject",
			Style:                "dotted",
			Healthy:              true,
		})
	}
	return edges, nil
}

// resourceExistenceIndex is a kind → {existence key} lookup built once per
// Match call, replacing what was an O(events × resources-of-that-kind)
// linear scan per event (hasResource/hasDeployment/hasReplicaSet/...) with
// an O(1) map lookup per event. Built fresh every call since bundle is
// rebuilt on every topology request — nothing here is cached across calls.
type resourceExistenceIndex map[string]map[string]struct{}

// existenceKey matches hasResource's original per-kind matching rule:
// Node and Namespace are cluster-scoped and were matched by name only
// (namespace ignored); every other kind requires an exact namespace+name
// match.
func existenceKey(kind, ns, name string) string {
	if kind == "Node" || kind == "Namespace" {
		return name
	}
	return ns + "/" + name
}

func (idx resourceExistenceIndex) add(kind, ns, name string) {
	set := idx[kind]
	if set == nil {
		set = make(map[string]struct{})
		idx[kind] = set
	}
	set[existenceKey(kind, ns, name)] = struct{}{}
}

func (idx resourceExistenceIndex) has(kind, ns, name string) bool {
	set, ok := idx[kind]
	if !ok {
		return false
	}
	_, ok = set[existenceKey(kind, ns, name)]
	return ok
}

// buildResourceExistenceIndex covers exactly the kinds hasResource used to
// switch on — any kind not listed here correctly reports !has(...), the
// same as hasResource's default case.
func buildResourceExistenceIndex(b *v2.ResourceBundle) resourceExistenceIndex {
	idx := make(resourceExistenceIndex, 13)
	for i := range b.Pods {
		idx.add("Pod", b.Pods[i].Namespace, b.Pods[i].Name)
	}
	for i := range b.Nodes {
		idx.add("Node", "", b.Nodes[i].Name)
	}
	for i := range b.Deployments {
		idx.add("Deployment", b.Deployments[i].Namespace, b.Deployments[i].Name)
	}
	for i := range b.ReplicaSets {
		idx.add("ReplicaSet", b.ReplicaSets[i].Namespace, b.ReplicaSets[i].Name)
	}
	for i := range b.StatefulSets {
		idx.add("StatefulSet", b.StatefulSets[i].Namespace, b.StatefulSets[i].Name)
	}
	for i := range b.DaemonSets {
		idx.add("DaemonSet", b.DaemonSets[i].Namespace, b.DaemonSets[i].Name)
	}
	for i := range b.Jobs {
		idx.add("Job", b.Jobs[i].Namespace, b.Jobs[i].Name)
	}
	for i := range b.CronJobs {
		idx.add("CronJob", b.CronJobs[i].Namespace, b.CronJobs[i].Name)
	}
	for i := range b.Services {
		idx.add("Service", b.Services[i].Namespace, b.Services[i].Name)
	}
	for i := range b.ConfigMaps {
		idx.add("ConfigMap", b.ConfigMaps[i].Namespace, b.ConfigMaps[i].Name)
	}
	for i := range b.Secrets {
		idx.add("Secret", b.Secrets[i].Namespace, b.Secrets[i].Name)
	}
	for i := range b.PVCs {
		idx.add("PersistentVolumeClaim", b.PVCs[i].Namespace, b.PVCs[i].Name)
	}
	for i := range b.Ingresses {
		idx.add("Ingress", b.Ingresses[i].Namespace, b.Ingresses[i].Name)
	}
	for i := range b.Namespaces {
		idx.add("Namespace", "", b.Namespaces[i].Name)
	}
	return idx
}
