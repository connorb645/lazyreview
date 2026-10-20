package components

import (
	"encoding/json"
	"slices"
	"strconv"
	"strings"

	"github.com/connorb645/lazyreview/internal/i18n"
	"github.com/connorb645/lazyreview/internal/models"
	"github.com/connorb645/lazyreview/internal/utils"
)

// Colors are the colours a component may be given, grey last, which
// context components take.
var Colors = []string{"blue", "teal", "purple", "amber", "olive", "rose", "indigo", "slate", "grey"}

const defaultMax = 40

var allowedKeys = map[string][]string{
	"root":         {"title", "summary", "areas", "components", "links", "summaries", "groups", "contextTypes"},
	"area":         {"id", "title", "description", "components"},
	"component":    {"id", "title", "description", "tag", "color", "group", "context", "paths", "packages"},
	"package":      {"id", "title", "description", "paths"},
	"link":         {"from", "to", "label", "kind"},
	"summary":      {"from", "to", "label"},
	"group":        {"id", "of", "title", "description"},
	"contextTypes": {"include", "exclude", "max"},
}

// Read checks the config raw holds, as JSON, and returns it with every
// optional part filled in, or an error listing every problem at once. With
// raw nil there is no config, and it returns the empty one.
func Read(tr *i18n.TranslationSet, raw []byte) (Config, error) {
	config := Config{ContextTypes: ContextTypes{Include: []string{}, Exclude: []string{}, Max: defaultMax}}
	var given any = jsonObject{values: map[string]any{}}
	if raw != nil {
		read, err := orderedJSON(raw)
		if err != nil {
			return config, err
		}
		given = read
	}
	c := &checker{tr: tr}
	root, ok := c.keys(given, allowedKeys["root"], tr.ComponentsRoot)
	if !ok {
		return config, c.failure()
	}
	config.Title = c.text(root, "title", tr.ComponentsRoot, false)
	config.Summary = c.text(root, "summary", tr.ComponentsRoot, false)

	componentIDs := map[string]bool{}
	for i, item := range c.list(root, "components", tr.ComponentsRoot) {
		label := c.named(tr.ComponentsComponent, item, i+1)
		if component, ok := c.component(item, label, componentIDs); ok {
			config.Components = append(config.Components, component)
		}
	}
	areaIDs, inArea := map[string]bool{}, map[string]bool{}
	for i, item := range c.list(root, "areas", tr.ComponentsRoot) {
		label := c.named(tr.ComponentsArea, item, i+1)
		object, ok := c.keys(item, allowedKeys["area"], label)
		if !ok {
			continue
		}
		area := Area{ID: c.id(object, label, areaIDs), Title: c.text(object, "title", label, true), Description: c.text(object, "description", label, false)}
		area.Components = c.strings(object, "components", label)
		if len(area.Components) == 0 {
			c.say(tr.ComponentsNeedsComponent, map[string]string{"label": label})
		}
		for _, m := range area.Components {
			if inArea[m] {
				c.say(tr.ComponentsInTwoAreas, map[string]string{"label": label, "component": m})
			}
			inArea[m] = true
		}
		area.ID = rawText(object, "id")
		config.Areas = append(config.Areas, area)
	}
	for i, item := range c.list(root, "links", tr.ComponentsRoot) {
		label := utils.ResolvePlaceholderString(tr.ComponentsLink, map[string]string{"number": strconv.Itoa(i + 1)})
		object, ok := c.keys(item, allowedKeys["link"], label)
		if !ok {
			continue
		}
		link := Link{From: c.text(object, "from", label, true), To: c.text(object, "to", label, true), Label: c.text(object, "label", label, true), Kind: "use"}
		if kind, held := object.values["kind"]; held {
			if kind != "use" && kind != "has" {
				c.say(tr.ComponentsBadKind, map[string]string{"label": label})
			} else {
				link.Kind = kind.(string)
			}
		}
		config.Links = append(config.Links, link)
	}
	for i, item := range c.list(root, "summaries", tr.ComponentsRoot) {
		label := utils.ResolvePlaceholderString(tr.ComponentsSummary, map[string]string{"number": strconv.Itoa(i + 1)})
		if object, ok := c.keys(item, allowedKeys["summary"], label); ok {
			c.text(object, "from", label, true)
			c.text(object, "to", label, true)
			config.Summaries = append(config.Summaries, Summary{From: rawText(object, "from"), To: rawText(object, "to"), Label: c.text(object, "label", label, true)})
		}
	}
	groupIDs := map[string]bool{}
	for i, item := range c.list(root, "groups", tr.ComponentsRoot) {
		label := c.named(tr.ComponentsGroup, item, i+1)
		if object, ok := c.keys(item, allowedKeys["group"], label); ok {
			config.Groups = append(config.Groups, Group{ID: c.id(object, label, groupIDs), Of: c.text(object, "of", label, true), Title: c.text(object, "title", label, true), Description: c.text(object, "description", label, false)})
		}
	}
	if given, held := root.values["contextTypes"]; held {
		if object, ok := c.keys(given, allowedKeys["contextTypes"], tr.ComponentsContextTypes); ok {
			if max, held := object.values["max"]; held {
				if n, whole := wholeNumber(max); whole {
					config.ContextTypes.Max = n
				} else {
					c.say(tr.ComponentsBadMax, nil)
				}
			}
			config.ContextTypes.Include = c.strings(object, "include", tr.ComponentsContextTypes)
			config.ContextTypes.Exclude = c.strings(object, "exclude", tr.ComponentsContextTypes)
		}
	}

	known := map[string]bool{"context": true}
	for id := range componentIDs {
		known[id] = true
	}
	for _, a := range config.Areas {
		for _, m := range a.Components {
			if !known[m] {
				c.say(tr.ComponentsAreaUnknown, map[string]string{"area": a.ID, "component": m})
			}
		}
	}
	for _, s := range config.Summaries {
		for _, end := range []string{s.From, s.To} {
			if !known[end] {
				c.say(tr.ComponentsSummaryUnknown, map[string]string{"component": end})
			}
		}
	}
	if len(c.problems) > 0 {
		return config, c.failure()
	}
	return config, nil
}

func (c *checker) component(item any, label string, seen map[string]bool) (Component, bool) {
	object, ok := c.keys(item, allowedKeys["component"], label)
	if !ok {
		return Component{}, false
	}
	component := Component{
		ID:          c.id(object, label, seen),
		Title:       c.text(object, "title", label, true),
		Description: c.text(object, "description", label, false),
		Tag:         c.text(object, "tag", label, false),
		Group:       c.text(object, "group", label, false),
	}
	if color, held := object.values["color"]; held {
		if name, ok := color.(string); ok && slices.Contains(Colors, name) {
			component.Color = name
		} else {
			c.say(c.tr.ComponentsBadColor, map[string]string{"label": label, "colors": strings.Join(Colors, ", ")})
		}
	}
	if context, held := object.values["context"]; held {
		if flag, ok := context.(bool); ok {
			component.Context = flag
		} else {
			c.say(c.tr.ComponentsBadContext, map[string]string{"label": label})
		}
	}
	component.Paths = c.strings(object, "paths", label)
	packageIDs := map[string]bool{}
	for j, p := range c.list(object, "packages", label) {
		plabel := utils.ResolvePlaceholderString(c.tr.ComponentsPackageOf, map[string]string{"component": label, "package": c.named(c.tr.ComponentsPackage, p, j+1)})
		pobject, ok := c.keys(p, allowedKeys["package"], plabel)
		if !ok {
			continue
		}
		pkg := Package{ID: c.id(pobject, plabel, packageIDs), Title: c.text(pobject, "title", plabel, true), Description: c.text(pobject, "description", plabel, false)}
		pkg.Paths = c.strings(pobject, "paths", plabel)
		if len(pkg.Paths) == 0 {
			c.say(c.tr.ComponentsNeedsPath, map[string]string{"label": plabel})
		}
		component.Packages = append(component.Packages, pkg)
	}
	if len(component.Paths) == 0 && len(component.Packages) == 0 {
		c.say(c.tr.ComponentsNeedsPaths, map[string]string{"label": label})
	}
	return component, true
}

func (c *checker) failure() error {
	listed := make([]string, len(c.problems))
	for i, p := range c.problems {
		listed[i] = utils.ResolvePlaceholderString(c.tr.ConfigListedProblem, map[string]string{"problem": p})
	}
	return models.Refused(utils.ResolvePlaceholderString(c.tr.ComponentsProblems, map[string]string{
		"count": utils.Plural(len(c.problems), c.tr.ProblemNoun), "problems": strings.Join(listed, "\n"),
	}))
}

func wholeNumber(v any) (int, bool) {
	n, ok := v.(json.Number)
	if !ok {
		return 0, false
	}
	f, err := n.Float64()
	if err != nil || f < 0 || f != float64(int(f)) {
		return 0, false
	}
	return int(f), true
}

func rawText(object jsonObject, key string) string {
	v, held := object.values[key]
	if !held {
		return "undefined"
	}
	return jsText(v)
}
