package inertia

type mergeSettings struct {
	appendRoot   bool
	prependRoot  bool
	deepMerge    bool
	appendPaths  []string
	prependPaths []string
	matchesOn    []string
}

func rootAppendMergeSettings() mergeSettings {
	return mergeSettings{appendRoot: true}
}

func (s mergeSettings) enabled() bool {
	return s.appendRoot || s.prependRoot || s.deepMerge || len(s.appendPaths) > 0 || len(s.prependPaths) > 0
}

func (s *mergeSettings) setAppend(paths []string) {
	*s = mergeSettings{matchesOn: append([]string(nil), s.matchesOn...)}
	if len(paths) == 0 {
		s.appendRoot = true
		return
	}
	s.appendPaths = append([]string(nil), paths...)
}

func (s *mergeSettings) setPrepend(paths []string) {
	*s = mergeSettings{matchesOn: append([]string(nil), s.matchesOn...)}
	if len(paths) == 0 {
		s.prependRoot = true
		return
	}
	s.prependPaths = append([]string(nil), paths...)
}

type mergeSettingsProvider interface {
	mergeSettings() mergeSettings
}
