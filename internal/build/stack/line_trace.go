package stack

func traceNew(steps []Step, path string, line int) trace {
	shown := []place{}
	for _, s := range steps {
		at, ok := back(s.FromTop, place{path: path, line: line})
		file, has := s.Own.ByNew[at.path]
		if !ok || !has {
			continue
		}
		step := TowardOld(file.Hunks, at.line)
		at.pull = s.Pull
		if step.Added {
			return trace{owner: &at}
		}
		if step.Shown {
			shown = append(shown, at)
		}
	}
	return trace{shown: shown}
}

func traceOld(steps []Step, path string, line int) trace {
	shown := []place{}
	for _, s := range steps {
		at, ok := forward(s.FromBase, place{path: path, line: line})
		file, has := s.Own.ByOld[at.path]
		if !ok || !has {
			continue
		}
		step := TowardNew(file.Hunks, at.line)
		target := place{pull: s.Pull, path: file.Path, line: at.line}
		if target.path == "" {
			target.path = at.path
		}
		if step.Removed {
			return trace{owner: &target}
		}
		if step.Shown {
			shown = append(shown, target)
		}
	}
	return trace{shown: shown}
}

func back(d StepDiff, at place) (place, bool) {
	file, ok := d.ByNew[at.path]
	if !ok {
		return at, true
	}
	step := TowardOld(file.Hunks, at.line)
	if step.Added {
		return place{}, false
	}
	if file.OldPath != "" {
		at.path = file.OldPath
	}
	at.line = step.Line
	return at, true
}

func forward(d StepDiff, at place) (place, bool) {
	file, ok := d.ByOld[at.path]
	if !ok {
		return at, true
	}
	step := TowardNew(file.Hunks, at.line)
	if step.Removed {
		return place{}, false
	}
	if file.Path != "" {
		at.path = file.Path
	}
	at.line = step.Line
	return at, true
}
