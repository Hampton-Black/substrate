package core

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	gitrepo "github.com/Hampton-Black/substrate/internal/git"
	sqlcdb "github.com/Hampton-Black/substrate/internal/db/sqlc"
)

var slugRE = regexp.MustCompile(`[^a-z0-9]+`)

// Service is the core entity layer and ACL chokepoint.
type Service struct {
	q    *sqlcdb.Queries
	conn *sql.DB
	git  *gitrepo.Repo
	log  *slog.Logger
}

// NewService constructs a Service from an open database and optional git repo.
func NewService(conn *sql.DB, git *gitrepo.Repo, log *slog.Logger) *Service {
	if log == nil {
		log = slog.Default()
	}
	return &Service{
		q:    sqlcdb.New(conn),
		conn: conn,
		git:  git,
		log:  log,
	}
}

// checkScope is the ACL chokepoint. Real enforcement lands in M5.
func (s *Service) checkScope(_callerPod, _entityPod string, _scope Scope) error {
	// TODO(spec): wire real ACLs in M5
	return nil
}

// RegisterPod registers a new pod; id is derived from name (slugified).
func (s *Service) RegisterPod(ctx context.Context, in RegisterPodInput) (Pod, error) {
	id := Slugify(in.Name)
	if id == "" {
		return Pod{}, fmt.Errorf("invalid pod name: %q", in.Name)
	}

	if err := s.checkScope("", id, ScopePod); err != nil {
		return Pod{}, err
	}

	now := time.Now().UTC()
	meta, _ := json.Marshal(map[string]any{})

	err := s.q.CreatePod(ctx, sqlcdb.CreatePodParams{
		ID:          id,
		DisplayName: in.Name,
		Owner:       in.Owner,
		PublicKey:   sql.NullString{},
		CreatedAt:   now,
		Active:      true,
		Metadata:    meta,
	})
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			return Pod{}, fmt.Errorf("pod %q already exists", id)
		}
		return Pod{}, fmt.Errorf("create pod: %w", err)
	}

	if err := s.emitEvent(ctx, id, "", "pod.registered", map[string]any{
		"pod_id": id,
		"name":   in.Name,
		"owner":  in.Owner,
	}); err != nil {
		return Pod{}, err
	}

	s.log.Info("pod registered", "pod_id", id)
	return s.GetPod(ctx, id)
}

// GetPod returns a pod by id.
func (s *Service) GetPod(ctx context.Context, id string) (Pod, error) {
	row, err := s.q.GetPod(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Pod{}, fmt.Errorf("pod not found: %s", id)
		}
		return Pod{}, fmt.Errorf("get pod: %w", err)
	}
	return podFromRow(row), nil
}

// ListPods returns all pods.
func (s *Service) ListPods(ctx context.Context) ([]Pod, error) {
	rows, err := s.q.ListPods(ctx)
	if err != nil {
		return nil, fmt.Errorf("list pods: %w", err)
	}
	out := make([]Pod, 0, len(rows))
	for _, r := range rows {
		out = append(out, podFromRow(r))
	}
	return out, nil
}

func (s *Service) emitEvent(ctx context.Context, podID, workstreamID, eventType string, payload map[string]any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal event payload: %w", err)
	}
	return s.q.InsertEvent(ctx, sqlcdb.InsertEventParams{
		PodID: sql.NullString{String: podID, Valid: podID != ""},
		WorkstreamID: sql.NullString{String: workstreamID, Valid: workstreamID != ""},
		EventType:    eventType,
		Payload:      body,
		OccurredAt:   time.Now().UTC(),
	})
}

// Slugify converts a display name to a pod id (lowercase kebab-case).
func Slugify(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	s = slugRE.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	return s
}

func podFromRow(r sqlcdb.Pod) Pod {
	p := Pod{
		ID:          r.ID,
		DisplayName: r.DisplayName,
		Owner:       r.Owner,
		CreatedAt:   r.CreatedAt,
		Active:      r.Active,
	}
	if r.PublicKey.Valid {
		p.PublicKey = r.PublicKey.String
	}
	if len(r.Metadata) > 0 {
		_ = json.Unmarshal(r.Metadata, &p.Metadata)
	}
	return p
}
