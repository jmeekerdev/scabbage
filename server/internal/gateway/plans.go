package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	"scabbage/internal/agent"
	pb "scabbage/internal/pb/scabbage/v1"
	"scabbage/internal/plan"
	"scabbage/internal/world"
)

const planHelp = "Plans: /plans lists them, /plan save NAME saves an agent's proposal on screen, " +
	"/plan share NAME, /plan unshare NAME, /plan rename OLD NEW, /plan delete NAME."

// isPlanBlock reports whether plans may use the block: anything players can place.
func isPlanBlock(name string) bool {
	b, ok := world.Named(name)
	return ok && b != world.Air && b != world.Bedrock && world.Placeable(b)
}

// tabBuild is build's arguments as the tab gets them, with a saved or proposed plan filled in.
type tabBuild struct {
	Plan     *plan.Plan `json:"plan"`
	X        *int       `json:"x,omitempty"`
	Y        *int       `json:"y,omitempty"`
	Z        *int       `json:"z,omitempty"`
	Facing   string     `json:"facing,omitempty"`
	Material string     `json:"material,omitempty"`
}

// propose shows the player a plan as a ghost, always waiting for their /yes, and keeps it for save_plan.
func (g *Gateway) propose(ctx context.Context, s *session, args json.RawMessage) (any, error) {
	var a struct {
		Plan     json.RawMessage `json:"plan"`
		X, Y, Z  *int
		Facing   string `json:"facing"`
		Material string `json:"material"`
	}
	if err := decodeArgs(args, &a); err != nil {
		return nil, err
	}
	p, err := g.resolvePlan(s.player.ID, a.Plan)
	if err != nil {
		return nil, err
	}
	out, err := s.agent.relay(ctx, s, agentCommand{g.agentIDs.Add(1), "build",
		mustJSON(tabBuild{&p, a.X, a.Y, a.Z, a.Facing, a.Material}), true})
	if err == nil {
		s.agent.proposed(p)
	}
	return out, err
}

// resolvePlan reads a plan sent whole and checks it, or loads a saved one named by a string.
func (g *Gateway) resolvePlan(owner int64, raw json.RawMessage) (plan.Plan, error) {
	var name string
	if json.Unmarshal(raw, &name) == nil {
		st, err := g.cfg.Plans.Get(owner, name)
		if errors.Is(err, plan.ErrNotFound) {
			return plan.Plan{}, agent.Refusal(fmt.Sprintf("there's no plan called %s; plans lists them", name))
		}
		return st.Plan, err
	}
	var p plan.Plan
	if len(raw) == 0 {
		return p, agent.Refusal("propose needs plan: a whole plan, or the name of a saved one")
	}
	if err := decodeArgs(raw, &p); err != nil {
		return p, err
	}
	if err := p.Check(isPlanBlock); err != nil {
		return p, agent.Refusal(err.Error())
	}
	return p, nil
}

// build relays a template build as it is, or fills in the saved plan it names.
func (g *Gateway) build(ctx context.Context, s *session, o agent.Owner, args json.RawMessage) (any, error) {
	var a struct {
		Template string `json:"template"`
		X, Y, Z  *int
		W, D, H  *int
		Facing   string `json:"facing"`
		Material string `json:"material"`
	}
	if err := decodeArgs(args, &a); err != nil {
		return nil, err
	}
	if slices.Contains(plan.Templates, strings.ToLower(a.Template)) {
		return s.agent.relay(ctx, s, agentCommand{g.agentIDs.Add(1), "build", args, o.MustConfirm()})
	}
	st, err := g.cfg.Plans.Get(o.PlayerID, a.Template)
	if errors.Is(err, plan.ErrNotFound) {
		return nil, agent.Refusal(fmt.Sprintf("there's no template or plan called %s; try %s, or one from plans",
			a.Template, strings.Join(plan.Templates, ", ")))
	}
	if err != nil {
		return nil, err
	}
	if a.W != nil || a.D != nil || a.H != nil {
		return nil, agent.Refusal("saved plans can't be resized; leave out w, d and h")
	}
	return s.agent.relay(ctx, s, agentCommand{g.agentIDs.Add(1), "build",
		mustJSON(tabBuild{&st.Plan, a.X, a.Y, a.Z, a.Facing, a.Material}), o.MustConfirm()})
}

// savePlan saves the plan last proposed in the tab, under its own name or a new one.
func (s *session) savePlan(args json.RawMessage) (any, error) {
	var a struct {
		Name string `json:"name"`
	}
	if err := decodeArgs(args, &a); err != nil {
		return nil, err
	}
	name, err := s.saveProposal(a.Name)
	if err != nil {
		return nil, agent.Refusal(err.Error())
	}
	return map[string]string{"saved": name}, nil
}

var errNoProposal = errors.New("propose a plan first; saving keeps the last one the player saw")

// saveProposal saves the last proposal as the player's, named name or else its own name.
func (s *session) saveProposal(name string) (string, error) {
	p, ok := s.agent.proposal()
	if !ok {
		return "", errNoProposal
	}
	if name != "" {
		p.Name = strings.ToLower(name)
	}
	if err := plan.CheckName(p.Name); err != nil {
		return "", err
	}
	return p.Name, s.gw.cfg.Plans.Save(s.player.ID, p)
}

// agentPlan is a plan as the plans and plan tools describe it.
type agentPlan struct {
	Name    string            `json:"name"`
	About   string            `json:"about,omitempty"`
	Size    string            `json:"size"`
	Cells   int               `json:"cells"`
	Owner   string            `json:"owner"`
	Shared  bool              `json:"shared"`
	Palette map[string]string `json:"palette,omitempty"`
	Layers  [][]string        `json:"layers,omitempty"`
}

func describePlan(st plan.Stored) agentPlan {
	return agentPlan{st.Name, st.About, fmt.Sprintf("%d×%d×%d", st.W, st.D, st.H), st.Count, st.Owner, st.Shared,
		st.Palette, st.Layers}
}

// listPlans is the owner's plans, then the shared ones.
func (g *Gateway) listPlans(o agent.Owner) (any, error) {
	all, err := g.cfg.Plans.List(o.PlayerID)
	out := make([]agentPlan, 0, len(all))
	for _, st := range all {
		out = append(out, describePlan(st))
	}
	return map[string]any{"plans": out}, err
}

// onePlan is a saved plan in full, so an agent can start from it.
func (g *Gateway) onePlan(o agent.Owner, args json.RawMessage) (any, error) {
	var a struct {
		Name string `json:"name"`
	}
	if err := decodeArgs(args, &a); err != nil {
		return nil, err
	}
	st, err := g.cfg.Plans.Get(o.PlayerID, a.Name)
	if errors.Is(err, plan.ErrNotFound) {
		return nil, agent.Refusal(fmt.Sprintf("there's no plan called %s; plans lists them", a.Name))
	}
	return describePlan(st), err
}

// requestPlan answers the tab's PlanRequest for /build.
func (s *session) requestPlan(ctx context.Context, m *pb.PlanRequest) error {
	reply := &pb.Plan{Name: m.Name}
	st, err := s.gw.cfg.Plans.Get(s.player.ID, m.Name)
	switch {
	case errors.Is(err, plan.ErrNotFound):
		reply.Error = fmt.Sprintf("There's no template or plan called %s. Try %s, or one from /plans.",
			m.Name, strings.Join(plan.Templates, ", "))
	case err != nil:
		return err
	default:
		reply.Json = string(mustJSON(st.Plan))
	}
	return s.write(ctx, &pb.ServerMessage{Body: &pb.ServerMessage_Plan{Plan: reply}})
}

// listPlansCommand runs /plans.
func (s *session) listPlansCommand(ctx context.Context) error {
	all, err := s.gw.cfg.Plans.List(s.player.ID)
	if err != nil {
		return err
	}
	var mine, shared []string
	for _, st := range all {
		size := fmt.Sprintf("%d×%d×%d", st.W, st.D, st.H)
		switch {
		case st.Owner != s.player.Name:
			shared = append(shared, fmt.Sprintf("%s (%s, by %s)", st.Name, size, st.Owner))
		case st.Shared:
			mine = append(mine, fmt.Sprintf("%s (%s, shared)", st.Name, size))
		default:
			mine = append(mine, fmt.Sprintf("%s (%s)", st.Name, size))
		}
	}
	if len(all) == 0 {
		return s.notice(ctx, "No plans yet. Have an agent propose one, then /plan save NAME.")
	}
	lines := []string{}
	if len(mine) > 0 {
		lines = append(lines, "Your plans: "+strings.Join(mine, ", ")+".")
	}
	if len(shared) > 0 {
		lines = append(lines, "Shared: "+strings.Join(shared, ", ")+".")
	}
	return s.notice(ctx, strings.Join(lines, " ")+" /build NAME builds one.")
}

// planCommand runs /plan save, share, unshare, rename and delete.
func (s *session) planCommand(ctx context.Context, arg string) error {
	words := strings.Fields(strings.ToLower(arg))
	if len(words) < 2 {
		return s.notice(ctx, planHelp)
	}
	svc, id, name := s.gw.cfg.Plans, s.player.ID, words[1]
	var err error
	var done string
	switch {
	case words[0] == "save":
		name, err = s.saveProposal(name)
		done = fmt.Sprintf("Saved %s. /build %s builds it.", name, name)
	case words[0] == "share":
		err = svc.Share(id, name, true)
		done = fmt.Sprintf("Shared %s: anyone can /build it now.", name)
	case words[0] == "unshare":
		err = svc.Share(id, name, false)
		done = fmt.Sprintf("%s is yours alone again.", name)
	case words[0] == "rename" && len(words) == 3:
		err = svc.Rename(id, name, words[2])
		done = fmt.Sprintf("Renamed %s to %s.", name, words[2])
	case words[0] == "delete":
		done, err = s.deletePlan(name)
	default:
		return s.notice(ctx, planHelp)
	}
	if err != nil {
		return s.notice(ctx, sentence(err))
	}
	return s.notice(ctx, done)
}

// deletePlan deletes the player's own plan, or for an admin, anyone's shared plan of that name.
func (s *session) deletePlan(name string) (string, error) {
	svc := s.gw.cfg.Plans
	err := svc.Delete(s.player.ID, name)
	if !errors.Is(err, plan.ErrNotFound) || !s.player.Admin {
		return fmt.Sprintf("Deleted %s.", name), err
	}
	st, err := svc.Get(s.player.ID, name)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("Deleted %s's shared plan %s.", st.Owner, name), svc.DeleteShared(name)
}

func mustJSON(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}
