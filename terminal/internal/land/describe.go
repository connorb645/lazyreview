package land

// Describe says what Land will do with plan: the commits it brings to
// main, then each push and the move of main here.
func Describe(plan Plan) []string {
	var steps []string
	for _, r := range plan.Remotes {
		if r.Main != plan.Commit {
			steps = append(steps, push(plan.Short, r.Name))
		}
	}
	if plan.Here != plan.Commit {
		steps = append(steps, move(plan.Short))
	}
	out := []string{landing(plan.Branch, plan.Short, len(plan.Adds))}
	for _, line := range plan.Adds {
		out = append(out, "  "+line)
	}
	if len(steps) > 0 {
		out = append(append(out, stepsHeading), steps...)
	}
	return out
}
