package config

// ordered is a YAMLMap of the pairs given, key then value, in their order.
func ordered(pairs ...any) YAMLMap {
	m := YAMLMap{Values: map[string]any{}}
	for i := 0; i+1 < len(pairs); i += 2 {
		key := pairs[i].(string)
		m.Keys = append(m.Keys, key)
		m.Values[key] = pairs[i+1]
	}
	return m
}

func joinedMap(maps ...YAMLMap) YAMLMap {
	out := YAMLMap{Values: map[string]any{}}
	for _, m := range maps {
		for _, k := range m.Keys {
			if _, held := out.Values[k]; !held {
				out.Keys = append(out.Keys, k)
			}
			out.Values[k] = m.Values[k]
		}
	}
	return out
}
