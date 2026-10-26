// notion-sync imports one selected roadmap and optionally polls it for changes.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/CaliLuke/go-plane/internal/notionsync"
)

type options struct {
	roadmap, team, origin, workspace, state, users, snapshot string
	apply, parents, roadmapOnly                              bool
	interval                                                 time.Duration
	requestInterval                                          time.Duration
}

func main() {
	var o options
	flag.StringVar(&o.roadmap, "roadmap", "", "Notion roadmap page URL or ID")
	flag.StringVar(&o.team, "team", "", "Notion team page URL or ID; requires a complete snapshot, instead of --roadmap")
	flag.StringVar(&o.origin, "vector", "http://localhost:8000", "Vector origin")
	flag.StringVar(&o.workspace, "workspace", "", "Target Vector workspace slug")
	flag.StringVar(&o.state, "state", "", "Private durable journal path; reuse for every run")
	flag.StringVar(&o.users, "users", "", "Optional JSON map of Notion user IDs to Vector member IDs")
	flag.StringVar(&o.snapshot, "snapshot", "", "Private normalized page snapshot from an authorized Notion connection (one-time import)")
	flag.BoolVar(&o.apply, "apply", false, "Apply changes (default is a read-only preview)")
	flag.BoolVar(&o.roadmapOnly, "roadmap-only", false, "Import only the roadmap; do not read linked projects or tasks")
	flag.BoolVar(&o.parents, "include-parents", false, "Include ancestor tasks, without importing their other children")
	flag.DurationVar(&o.interval, "interval", 0, "Repeat interval, at least 5m; requires --apply")
	flag.DurationVar(&o.requestInterval, "request-interval", 1100*time.Millisecond, "Delay between Vector requests; must respect the server's configured rate limit")
	flag.Parse()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := execute(ctx, o); err != nil && !errors.Is(err, context.Canceled) {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func execute(ctx context.Context, o options) error {
	selection := o.roadmap
	if o.team != "" {
		if o.roadmap != "" || o.snapshot == "" || o.parents || o.roadmapOnly {
			return fmt.Errorf("--team requires --snapshot and cannot combine with --roadmap, --roadmap-only, or --include-parents")
		}
		selection = o.team
	}
	root, err := notionsync.PageID(selection)
	if err != nil {
		return err
	}
	if o.team != "" {
		o.team = root
	} else {
		o.roadmap = root
	}
	if o.interval != 0 && (!o.apply || o.interval < 5*time.Minute) {
		return fmt.Errorf("--interval requires --apply and at least 5m")
	}
	notion, err := sourceFor(o)
	if err != nil {
		return err
	}
	vector, err := notionsync.NewVector(o.origin, o.workspace, os.Getenv("VECTOR_API_KEY"))
	if err != nil {
		return err
	}
	if o.requestInterval != 0 {
		if err = vector.SetRequestInterval(o.requestInterval); err != nil {
			return err
		}
	}
	users := map[string]string{}
	if o.users != "" {
		data, readErr := os.ReadFile(o.users)
		if readErr != nil {
			return readErr
		}
		if err = json.Unmarshal(data, &users); err != nil {
			return err
		}
	}
	for {
		if err = syncOnce(ctx, o, notion, vector, users); err != nil {
			return err
		}
		if o.interval == 0 {
			return nil
		}
		timer := time.NewTimer(o.interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}

func sourceFor(o options) (notionsync.Source, error) {
	if o.snapshot == "" {
		return notionsync.NewNotion(os.Getenv("NOTION_TOKEN"), !o.roadmapOnly)
	}
	if o.interval != 0 {
		return nil, fmt.Errorf("--snapshot cannot be polled; export fresh pages for each import")
	}
	file, err := os.Open(o.snapshot)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()
	return notionsync.ReadSnapshot(file)
}

func syncOnce(ctx context.Context, o options, source notionsync.Source, vector *notionsync.Vector, users map[string]string) error {
	selection := o.roadmap
	if o.team != "" {
		selection = "team:" + o.team
	}
	scope := strings.Join([]string{vector.Origin, vector.Workspace, selection}, "|")
	journal, err := notionsync.OpenJournal(o.state, scope)
	if err != nil {
		return err
	}
	defer func() { _ = journal.Close() }()
	graph, err := discover(ctx, o, source)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "Selected %d modules and %d work items (apply=%t).\n", len(graph.Modules), len(graph.Tasks), o.apply)
	runner := notionsync.Runner{Vector: vector, Journal: journal, Apply: o.apply, Users: users}
	report, runErr := runner.Run(ctx, graph)
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err = encoder.Encode(report); err != nil {
		return err
	}
	if runErr != nil {
		return runErr
	}
	if len(report.Conflicts) > 0 {
		return fmt.Errorf("sync has conflicts; review the reported fields before resuming")
	}
	return nil
}

func discover(ctx context.Context, o options, source notionsync.Source) (notionsync.Graph, error) {
	if o.team != "" {
		snapshot, ok := source.(notionsync.Snapshot)
		if !ok {
			return notionsync.Graph{}, fmt.Errorf("team import requires a complete snapshot")
		}
		return notionsync.DiscoverTeam(ctx, snapshot, o.team)
	}
	return notionsync.Discover(ctx, source, o.roadmap, o.parents, o.roadmapOnly)
}
