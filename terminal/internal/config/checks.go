package config

import (
	"math"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/connorb645/lazyreview/terminal/internal/i18n"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

const pullColours = 8

// check is a value given for a setting checked: what it reads as, and every
// problem with it, in the config's words.
func check(tr *i18n.TranslationSet, s *Setting, key string, value any) (any, []string) {
	say := func(words string, extra ...string) []string {
		fields := map[string]string{"key": key, "value": Shown(value)}
		for i := 0; i+1 < len(extra); i += 2 {
			fields[extra[i]] = extra[i+1]
		}
		return []string{utils.ResolvePlaceholderString(words, fields)}
	}
	text, isText := value.(string)
	switch s.Check {
	case "folder":
		if isText && (text == "~" || strings.HasPrefix(text, "~/") || filepath.IsAbs(text)) {
			return text, nil
		}
		return nil, say(tr.ConfigNotAFolder)
	case "port":
		if n, ok := whole(value); ok && n >= 1 && n <= 65535 {
			return float64(n), nil
		}
		return nil, say(tr.ConfigNotAPort)
	case "command":
		if isText && strings.TrimSpace(text) != "" {
			return text, nil
		}
		return nil, say(tr.ConfigNotACommand)
	case "text":
		if isText {
			return text, nil
		}
		return nil, say(tr.ConfigNotText)
	case "choice":
		for _, c := range s.Choices {
			if isText && c == text {
				return text, nil
			}
		}
		return nil, say(tr.ConfigNotAChoice, "choices", strings.Join(s.Choices[:len(s.Choices)-1], ", "), "last", s.Choices[len(s.Choices)-1])
	case "accounts":
		return checkAccounts(tr, value, key)
	case "repositories":
		if value == nil {
			return []string{}, nil
		}
		list, ok := value.([]any)
		if !ok {
			return nil, say(tr.ConfigNotRepositories)
		}
		var names []string
		var problems []string
		for _, item := range list {
			name, ok := item.(string)
			if !ok || !repositoryPattern.MatchString(name) {
				problems = append(problems, utils.ResolvePlaceholderString(tr.ConfigNotARepository, map[string]string{"key": key, "value": Shown(item)}))
				continue
			}
			names = append(names, name)
		}
		if problems != nil {
			return nil, problems
		}
		return append([]string{}, names...), nil
	case "colour":
		if c, ok := colourOf(value); ok {
			return c, nil
		}
		return nil, say(tr.ConfigNotAColour)
	case "colours":
		list, ok := value.([]any)
		if ok && len(list) == pullColours {
			out := make([]string, len(list))
			for i, item := range list {
				if out[i], ok = colourOf(item); !ok {
					break
				}
			}
			if ok {
				return out, nil
			}
		}
		return nil, say(tr.ConfigNotColours, "count", strconv.Itoa(pullColours))
	case "keys":
		labels := []any{value}
		if list, ok := value.([]any); ok {
			labels = list
		}
		why := ""
		if list, ok := value.([]any); ok && len(list) == 0 {
			why = tr.ConfigKeyEmpty
		}
		for _, l := range labels {
			if why == "" {
				why = LabelProblem(tr, l)
			}
		}
		if why != "" {
			return nil, say(tr.ConfigNotKeys, "why", why)
		}
		if list, ok := value.([]any); ok {
			return texts(list), nil
		}
		return value, nil
	case "numberedKeys":
		list, ok := value.([]any)
		count := len(s.Default.([]string))
		if ok && len(list) == count {
			for _, l := range list {
				ok = ok && LabelProblem(tr, l) == ""
			}
			if ok {
				return texts(list), nil
			}
		}
		return nil, say(tr.ConfigNotNumberedKeys, "count", strconv.Itoa(count))
	}
	return nil, say(tr.ConfigNotText)
}

func checkAccounts(tr *i18n.TranslationSet, value any, key string) (any, []string) {
	list, ok := value.([]any)
	if !ok {
		return nil, []string{utils.ResolvePlaceholderString(tr.ConfigNotAccounts, map[string]string{"key": key, "value": Shown(value)})}
	}
	var problems []string
	say := func(words string, fields map[string]string) {
		problems = append(problems, utils.ResolvePlaceholderString(words, fields))
	}
	logins := map[string]bool{}
	accounts := []Account{}
	for i, item := range list {
		account, isMap := item.(YAMLMap)
		login, _ := account.Values["login"].(string)
		if !isMap || !loginPattern.MatchString(login) {
			say(tr.ConfigNoLogin, map[string]string{"number": strconv.Itoa(i + 1)})
			continue
		}
		for _, field := range account.Keys {
			if field != "login" && field != "teams" {
				say(tr.ConfigAccountKey, map[string]string{"login": login, "key": field})
			}
		}
		if logins[login] {
			say(tr.ConfigLoginTwice, map[string]string{"login": login})
		}
		logins[login] = true
		given := account.Values["teams"]
		if given == nil {
			given = []any{}
		}
		teamList, ok := given.([]any)
		if !ok {
			say(tr.ConfigTeamsNotList, map[string]string{"login": login})
			continue
		}
		read := Account{Login: login, Teams: []Team{}}
		for _, t := range teamList {
			team, isMap := t.(YAMLMap)
			name, _ := team.Values["name"].(string)
			if !isMap || !teamPattern.MatchString(name) {
				say(tr.ConfigTeamNotNamed, map[string]string{"login": login})
				continue
			}
			for _, field := range team.Keys {
				if field != "name" && field != "since" {
					say(tr.ConfigTeamKey, map[string]string{"team": name, "key": field})
				}
			}
			var since *string
			if given, held := team.Values["since"]; held {
				text, isText := given.(string)
				if !isText || !datePattern.MatchString(text) {
					say(tr.ConfigSinceNotDate, map[string]string{"team": name})
				} else {
					since = &text
				}
			}
			read.Teams = append(read.Teams, Team{Name: name, Since: since})
		}
		accounts = append(accounts, read)
	}
	if problems != nil {
		return nil, problems
	}
	return accounts, nil
}

func colourOf(v any) (string, bool) {
	if text, ok := v.(string); ok && colourPattern.MatchString(text) {
		return text, true
	}
	if n, ok := whole(v); ok && n >= 0 && n <= 255 {
		return strconv.Itoa(n), true
	}
	return "", false
}

func whole(v any) (int, bool) {
	f, ok := v.(float64)
	if !ok || math.IsInf(f, 0) || math.IsNaN(f) || f != math.Trunc(f) || math.Abs(f) > 1<<53 {
		return 0, false
	}
	return int(f), true
}

func texts(list []any) []string {
	out := make([]string, len(list))
	for i, item := range list {
		out[i], _ = item.(string)
	}
	return out
}
