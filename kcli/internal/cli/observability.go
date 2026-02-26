package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

type k8sEventList struct {
	Items []k8sEvent `json:"items"`
}

type k8sEvent struct {
	Type      string `json:"type"`
	Reason    string `json:"reason"`
	Message   string `json:"message"`
	Count     int    `json:"count"`
	EventTime string `json:"eventTime"`

	FirstTimestamp string `json:"firstTimestamp"`
	LastTimestamp  string `json:"lastTimestamp"`

	InvolvedObject struct {
		Kind      string `json:"kind"`
		Name      string `json:"name"`
		Namespace string `json:"namespace"`
	} `json:"involvedObject"`

	Metadata struct {
		Name              string `json:"name"`
		Namespace         string `json:"namespace"`
		CreationTimestamp string `json:"creationTimestamp"`
	} `json:"metadata"`
}

type eventRecord struct {
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"`
	Namespace string    `json:"namespace"`
	Object    string    `json:"object"`
	Reason    string    `json:"reason"`
	Message   string    `json:"message"`
	Count     int       `json:"count,omitempty"`
}

type podHealthSummary struct {
	Total         int
	Running       int
	Pending       int
	Failed        int
	Succeeded     int
	CrashLoop     int
	TotalRestarts int
	RestartPods   int
}

type nodeHealthSummary struct {
	Total       int
	Ready       int
	NotReady    int
	MemoryPress int
	DiskPress   int
	PIDPress    int
}

func newMetricsCmd(a *app) *cobra.Command {
	return &cobra.Command{
		Use:                "metrics [resource]",
		Short:              "Top/metrics view for resources",
		GroupID:            "observability",
		DisableFlagParsing: true,
		RunE: func(_ *cobra.Command, args []string) error {
			if len(args) == 0 {
				return a.runKubectl([]string{"top", "pods", "-A"})
			}
			return a.runKubectl(append([]string{"top"}, args...))
		},
	}
}

func newHealthCmd(a *app) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "health [pods|nodes]",
		Short:   "Cluster and resource health summary",
		GroupID: "observability",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return printOverallHealth(a, cmd)
			}
			switch strings.ToLower(strings.TrimSpace(args[0])) {
			case "pods", "pod":
				s, err := fetchPodHealthSummary(cmd.Context(), a)
				if err != nil {
					return err
				}
				printPodHealthSummary(cmd, s)
				return nil
			case "nodes", "node":
				s, err := fetchNodeHealthSummary(cmd.Context(), a)
				if err != nil {
					return err
				}
				printNodeHealthSummary(cmd, s)
				return nil
			default:
				return fmt.Errorf("unsupported health target %q (use pods|nodes)", args[0])
			}
		},
	}
	return cmd
}

func printOverallHealth(a *app, cmd *cobra.Command) error {
	pods, err := fetchPodHealthSummary(cmd.Context(), a)
	if err != nil {
		return err
	}
	nodes, err := fetchNodeHealthSummary(cmd.Context(), a)
	if err != nil {
		return err
	}
	score := healthScore(pods, nodes)
	fmt.Fprintf(cmd.OutOrStdout(), "Health Score: %d/100\n", score)
	printPodHealthSummary(cmd, pods)
	printNodeHealthSummary(cmd, nodes)
	return nil
}

func newRestartsCmd(a *app) *cobra.Command {
	var recent string
	var threshold int
	var output string
	cmd := &cobra.Command{
		Use:     "restarts",
		Short:   "List pods sorted by restart count",
		GroupID: "observability",
		RunE: func(c *cobra.Command, _ []string) error {
			pods, err := fetchPods(c.Context(), a)
			if err != nil {
				return err
			}
			cutoff := time.Time{}
			if strings.TrimSpace(recent) != "" {
				d, err := time.ParseDuration(strings.TrimSpace(recent))
				if err != nil {
					return fmt.Errorf("invalid --recent value %q: %w", recent, err)
				}
				cutoff = time.Now().Add(-d)
			}
			records := buildRestartRecords(pods, threshold, cutoff)
			sort.SliceStable(records, func(i, j int) bool { return records[i].Restarts > records[j].Restarts })
			switch strings.ToLower(strings.TrimSpace(output)) {
			case "table", "":
				printRestartTable(c, records)
				return nil
			case "json":
				b, err := json.MarshalIndent(records, "", "  ")
				if err != nil {
					return err
				}
				fmt.Fprintln(c.OutOrStdout(), string(b))
				return nil
			default:
				return fmt.Errorf("unsupported --output %q (supported: table, json)", output)
			}
		},
	}
	cmd.Flags().StringVar(&recent, "recent", "", "only include pods with recent restarts in this window (e.g. 1h)")
	cmd.Flags().IntVar(&threshold, "threshold", 1, "minimum restart count to include")
	cmd.Flags().StringVar(&output, "output", "table", "output format: table|json")
	return cmd
}

type restartRecord struct {
	Namespace string    `json:"namespace"`
	Name      string    `json:"name"`
	Node      string    `json:"node"`
	Phase     string    `json:"phase"`
	Restarts  int       `json:"restarts"`
	LastAt    time.Time `json:"lastRestartTime,omitempty"`
}

func buildRestartRecords(list *k8sPodList, threshold int, cutoff time.Time) []restartRecord {
	if list == nil {
		return nil
	}
	if threshold <= 0 {
		threshold = 1
	}
	out := make([]restartRecord, 0, len(list.Items))
	for _, p := range list.Items {
		total := 0
		last := time.Time{}
		for _, cs := range p.Status.ContainerStatuses {
			total += cs.RestartCount
			if ts := parseRFC3339(cs.LastState.Terminated.FinishedAt); ts.After(last) {
				last = ts
			}
		}
		if total < threshold {
			continue
		}
		if !cutoff.IsZero() && !last.IsZero() && last.Before(cutoff) {
			continue
		}
		out = append(out, restartRecord{
			Namespace: p.Metadata.Namespace,
			Name:      p.Metadata.Name,
			Node:      p.Spec.NodeName,
			Phase:     p.Status.Phase,
			Restarts:  total,
			LastAt:    last,
		})
	}
	return out
}

func printRestartTable(cmd *cobra.Command, records []restartRecord) {
	if len(records) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No restarted pods found.")
		return
	}
	fmt.Fprintf(cmd.OutOrStdout(), "%-18s %-38s %-10s %-9s %-20s\n", "NAMESPACE", "NAME", "RESTARTS", "PHASE", "LAST")
	for _, r := range records {
		last := "-"
		if !r.LastAt.IsZero() {
			last = r.LastAt.Format("2006-01-02 15:04:05")
		}
		fmt.Fprintf(cmd.OutOrStdout(), "%-18s %-38s %-10d %-9s %-20s\n", truncateCell(r.Namespace, 18), truncateCell(r.Name, 38), r.Restarts, truncateCell(r.Phase, 9), last)
	}
}

func newInstabilityCmd(a *app) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "instability",
		Short:   "Quick instability snapshot (restarts + warning events)",
		GroupID: "observability",
		RunE: func(cmd *cobra.Command, _ []string) error {
			fmt.Fprintln(cmd.OutOrStdout(), "== Restart Leaders ==")
			pods, err := fetchPods(cmd.Context(), a)
			if err != nil {
				return err
			}
			printRestartTable(cmd, buildRestartRecords(pods, 1, time.Time{}))

			fmt.Fprintln(cmd.OutOrStdout(), "\n== Recent Warning Events ==")
			records, err := fetchEvents(cmd.Context(), a)
			if err != nil {
				return err
			}
			warnings := filterEventsByType(records, "Warning")
			sort.SliceStable(warnings, func(i, j int) bool { return warnings[i].Timestamp.After(warnings[j].Timestamp) })
			if len(warnings) > 25 {
				warnings = warnings[:25]
			}
			printEventTable(cmd, warnings)
			return nil
		},
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "pods",
		Short: "Pod-only instability summary (restart leaders)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			pods, err := fetchPods(cmd.Context(), a)
			if err != nil {
				return err
			}
			records := buildRestartRecords(pods, 1, time.Time{})
			sort.SliceStable(records, func(i, j int) bool { return records[i].Restarts > records[j].Restarts })
			printRestartTable(cmd, records)
			return nil
		},
	})
	return cmd
}

func newEventsCmd(a *app) *cobra.Command {
	var recent string
	var output string
	var includeAll bool
	var evType string
	var watch bool
	cmd := &cobra.Command{
		Use:     "events",
		Short:   "View cluster events",
		GroupID: "observability",
		RunE: func(c *cobra.Command, _ []string) error {
			if watch {
				args := []string{"get", "events", "-A", "--watch"}
				if strings.TrimSpace(evType) != "" {
					args = append(args, "--field-selector", "type="+strings.TrimSpace(evType))
				}
				return a.runKubectl(args)
			}
			records, err := fetchEvents(c.Context(), a)
			if err != nil {
				return err
			}
			if !includeAll {
				window := 1 * time.Hour
				if strings.TrimSpace(recent) != "" {
					d, err := time.ParseDuration(strings.TrimSpace(recent))
					if err != nil {
						return fmt.Errorf("invalid --recent value %q: %w", recent, err)
					}
					window = d
				}
				records = filterEventsByRecent(records, window, time.Now())
			}
			if strings.TrimSpace(evType) != "" {
				records = filterEventsByType(records, evType)
			}
			sort.SliceStable(records, func(i, j int) bool { return records[i].Timestamp.After(records[j].Timestamp) })
			switch strings.ToLower(strings.TrimSpace(output)) {
			case "table", "":
				printEventTable(c, records)
				return nil
			case "json":
				return printEventsJSON(c, records)
			default:
				return fmt.Errorf("unsupported --output %q (supported: table, json)", output)
			}
		},
	}
	cmd.Flags().StringVar(&recent, "recent", "", "only show events within this duration window (e.g. 30m, 2h); defaults to 1h unless --all")
	cmd.Flags().StringVar(&output, "output", "table", "output format: table|json")
	cmd.Flags().BoolVar(&includeAll, "all", false, "show all events without recent time filter")
	cmd.Flags().StringVar(&evType, "type", "", "event type filter (e.g. Warning, Normal)")
	cmd.Flags().BoolVar(&watch, "watch", false, "watch events stream")
	return cmd
}

func fetchPodHealthSummary(ctx context.Context, a *app) (podHealthSummary, error) {
	list, err := fetchPods(ctx, a)
	if err != nil {
		return podHealthSummary{}, err
	}
	s := podHealthSummary{Total: len(list.Items)}
	for _, p := range list.Items {
		switch strings.ToLower(strings.TrimSpace(p.Status.Phase)) {
		case "running":
			s.Running++
		case "pending":
			s.Pending++
		case "failed":
			s.Failed++
		case "succeeded":
			s.Succeeded++
		}
		totalRestarts := 0
		for _, cs := range p.Status.ContainerStatuses {
			totalRestarts += cs.RestartCount
			if strings.EqualFold(cs.State.Waiting.Reason, "CrashLoopBackOff") {
				s.CrashLoop++
			}
		}
		s.TotalRestarts += totalRestarts
		if totalRestarts > 0 {
			s.RestartPods++
		}
	}
	return s, nil
}

func fetchNodeHealthSummary(ctx context.Context, a *app) (nodeHealthSummary, error) {
	list, err := fetchNodes(ctx, a)
	if err != nil {
		return nodeHealthSummary{}, err
	}
	s := nodeHealthSummary{Total: len(list.Items)}
	for _, n := range list.Items {
		ready := false
		for _, c := range n.Status.Conditions {
			t := strings.TrimSpace(c.Type)
			st := strings.EqualFold(strings.TrimSpace(c.Status), "True")
			switch t {
			case "Ready":
				ready = st
			case "MemoryPressure":
				if st {
					s.MemoryPress++
				}
			case "DiskPressure":
				if st {
					s.DiskPress++
				}
			case "PIDPressure":
				if st {
					s.PIDPress++
				}
			}
		}
		if ready {
			s.Ready++
		} else {
			s.NotReady++
		}
	}
	return s, nil
}

func healthScore(pods podHealthSummary, nodes nodeHealthSummary) int {
	score := 100
	if nodes.Total > 0 {
		nodePenalty := int(float64(nodes.NotReady) / float64(nodes.Total) * 60)
		score -= nodePenalty
	}
	score -= minInt(20, pods.CrashLoop*3)
	score -= minInt(12, pods.RestartPods)
	score -= minInt(8, nodes.MemoryPress+nodes.DiskPress+nodes.PIDPress)
	if score < 0 {
		return 0
	}
	if score > 100 {
		return 100
	}
	return score
}

func printPodHealthSummary(cmd *cobra.Command, s podHealthSummary) {
	fmt.Fprintln(cmd.OutOrStdout(), "\nPods:")
	fmt.Fprintf(cmd.OutOrStdout(), "  total=%d running=%d pending=%d failed=%d succeeded=%d\n", s.Total, s.Running, s.Pending, s.Failed, s.Succeeded)
	fmt.Fprintf(cmd.OutOrStdout(), "  restartPods=%d totalRestarts=%d crashLoop=%d\n", s.RestartPods, s.TotalRestarts, s.CrashLoop)
}

func printNodeHealthSummary(cmd *cobra.Command, s nodeHealthSummary) {
	fmt.Fprintln(cmd.OutOrStdout(), "\nNodes:")
	fmt.Fprintf(cmd.OutOrStdout(), "  total=%d ready=%d notReady=%d\n", s.Total, s.Ready, s.NotReady)
	fmt.Fprintf(cmd.OutOrStdout(), "  pressure(memory=%d disk=%d pid=%d)\n", s.MemoryPress, s.DiskPress, s.PIDPress)
}

func fetchEvents(ctx context.Context, a *app) ([]eventRecord, error) {
	out, err := a.captureKubectlCtx(ctx, []string{"get", "events", "-A", "-o", "json"})
	if err != nil {
		return nil, err
	}
	var list k8sEventList
	if err := json.Unmarshal([]byte(out), &list); err != nil {
		return nil, fmt.Errorf("failed to parse events JSON: %w", err)
	}
	records := make([]eventRecord, 0, len(list.Items))
	for _, item := range list.Items {
		ts := parseEventTime(item)
		ns := strings.TrimSpace(item.InvolvedObject.Namespace)
		if ns == "" {
			ns = strings.TrimSpace(item.Metadata.Namespace)
		}
		obj := strings.TrimSpace(item.InvolvedObject.Kind + "/" + item.InvolvedObject.Name)
		if obj == "/" {
			obj = "-"
		}
		records = append(records, eventRecord{
			Timestamp: ts,
			Type:      strings.TrimSpace(item.Type),
			Namespace: ns,
			Object:    obj,
			Reason:    strings.TrimSpace(item.Reason),
			Message:   strings.TrimSpace(item.Message),
			Count:     item.Count,
		})
	}
	return records, nil
}

func parseEventTime(e k8sEvent) time.Time {
	candidates := []string{e.LastTimestamp, e.EventTime, e.FirstTimestamp, e.Metadata.CreationTimestamp}
	for _, raw := range candidates {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		if t, err := time.Parse(time.RFC3339, raw); err == nil {
			return t
		}
	}
	return time.Time{}
}

func filterEventsByRecent(records []eventRecord, window time.Duration, now time.Time) []eventRecord {
	if window <= 0 {
		return records
	}
	cutoff := now.Add(-window)
	out := make([]eventRecord, 0, len(records))
	for _, r := range records {
		if !r.Timestamp.IsZero() && r.Timestamp.Before(cutoff) {
			continue
		}
		out = append(out, r)
	}
	return out
}

func filterEventsByType(records []eventRecord, evType string) []eventRecord {
	t := strings.ToLower(strings.TrimSpace(evType))
	if t == "" {
		return records
	}
	out := make([]eventRecord, 0, len(records))
	for _, r := range records {
		if strings.EqualFold(strings.TrimSpace(r.Type), t) {
			out = append(out, r)
		}
	}
	return out
}

func parseRFC3339(raw string) time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}
	}
	t, _ := time.Parse(time.RFC3339, raw)
	return t
}

func printEventTable(cmd *cobra.Command, records []eventRecord) {
	if len(records) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No events found.")
		return
	}
	fmt.Fprintf(cmd.OutOrStdout(), "%-20s %-8s %-18s %-30s %-18s %s\n", "TIME", "TYPE", "NAMESPACE", "OBJECT", "REASON", "MESSAGE")
	for _, r := range records {
		ts := "-"
		if !r.Timestamp.IsZero() {
			ts = r.Timestamp.Format("2006-01-02 15:04:05")
		}
		msg := r.Message
		if len(msg) > 120 {
			msg = msg[:117] + "..."
		}
		fmt.Fprintf(
			cmd.OutOrStdout(),
			"%-20s %-8s %-18s %-30s %-18s %s\n",
			ts,
			emptyDash(r.Type),
			emptyDash(r.Namespace),
			truncateCell(r.Object, 30),
			truncateCell(r.Reason, 18),
			msg,
		)
	}
}

func printEventsJSON(cmd *cobra.Command, records []eventRecord) error {
	b, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintln(cmd.OutOrStdout(), string(b))
	return nil
}

func truncateCell(v string, limit int) string {
	if len(v) <= limit {
		return emptyDash(v)
	}
	if limit <= 3 {
		return v[:limit]
	}
	return v[:limit-3] + "..."
}

func emptyDash(v string) string {
	if strings.TrimSpace(v) == "" {
		return "-"
	}
	return v
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
