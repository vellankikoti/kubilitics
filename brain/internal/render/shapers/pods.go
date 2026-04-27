// Package shapers contains per-tool transforms from raw MCP tool
// output to wire-shape render data. This is deterministic — the LLM
// is never involved.
package shapers

import (
	"encoding/json"
	"fmt"
	"time"
)

// Shapers is the registry of tool name → shaper function. Every
// Deterministic tool in render.registry MUST have an entry here
// (enforced by an architecture test in package render).
var Shapers = map[string]func(json.RawMessage) (json.RawMessage, error){
	"list_pods":    ShapeListPods,
	"get_pod_yaml": ShapeGetPodYaml,
}

type column struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Align string `json:"align,omitempty"`
}

type table struct {
	Columns []column                 `json:"columns"`
	Rows    []map[string]interface{} `json:"rows"`
}

type podLite struct {
	Metadata struct {
		Name              string    `json:"name"`
		Namespace         string    `json:"namespace"`
		CreationTimestamp time.Time `json:"creationTimestamp"`
	} `json:"metadata"`
	Spec struct {
		Containers []struct{ Name string `json:"name"` } `json:"containers"`
	} `json:"spec"`
	Status struct {
		Phase             string `json:"phase"`
		ContainerStatuses []struct {
			Ready        bool `json:"ready"`
			RestartCount int  `json:"restartCount"`
		} `json:"containerStatuses"`
	} `json:"status"`
}

func ShapeListPods(raw json.RawMessage) (json.RawMessage, error) {
	var pods []podLite
	if err := json.Unmarshal(raw, &pods); err != nil {
		return nil, fmt.Errorf("list_pods shaper: %w", err)
	}
	t := table{
		Columns: []column{
			{Key: "NAME", Label: "NAME"},
			{Key: "READY", Label: "READY"},
			{Key: "STATUS", Label: "STATUS"},
			{Key: "RESTARTS", Label: "RESTARTS", Align: "right"},
			{Key: "AGE", Label: "AGE"},
		},
		Rows: make([]map[string]interface{}, len(pods)),
	}
	for i, p := range pods {
		ready := 0
		restarts := 0
		for _, cs := range p.Status.ContainerStatuses {
			if cs.Ready { ready++ }
			restarts += cs.RestartCount
		}
		t.Rows[i] = map[string]interface{}{
			"NAME":     p.Metadata.Name,
			"READY":    fmt.Sprintf("%d/%d", ready, len(p.Spec.Containers)),
			"STATUS":   p.Status.Phase,
			"RESTARTS": restarts,
			"AGE":      humanAge(p.Metadata.CreationTimestamp),
		}
	}
	return json.Marshal(t)
}

func humanAge(t time.Time) string {
	if t.IsZero() { return "?" }
	d := time.Since(t)
	if d < 0 { return "?" } // future timestamp (clock skew); avoid negative duration strings
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}

// ShapeGetPodYaml is a passthrough — the tool already returns
// {"yaml": "..."}. We re-marshal to enforce the wire shape.
func ShapeGetPodYaml(raw json.RawMessage) (json.RawMessage, error) {
	var in struct{ Yaml string `json:"yaml"` }
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, fmt.Errorf("get_pod_yaml shaper: %w", err)
	}
	out, err := json.Marshal(struct {
		Yaml string `json:"yaml"`
	}{Yaml: in.Yaml})
	return out, err
}
