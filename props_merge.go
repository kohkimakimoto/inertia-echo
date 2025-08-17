package inertia

// Mergeable represents a prop that can be merged
type Mergeable interface {
	ShouldMerge() bool
	ShouldDeepMerge() bool
	MatchesOn() []string
}

type MergeProp struct {
	value     any
	deepMerge bool
	matchesOn []string
}

func (p *MergeProp) ShouldMerge() bool {
	return true
}

func (p *MergeProp) DeepMerge() {
	p.deepMerge = true
}

func (p *MergeProp) ShouldDeepMerge() bool {
	return p.deepMerge
}

func (p *MergeProp) MatchesOn() []string {
	return p.matchesOn
}

func (p *MergeProp) MatchOn(fields ...string) *MergeProp {
	p.matchesOn = append(p.matchesOn, fields...)
	return p
}

func Merge(value any) *MergeProp {
	return &MergeProp{
		value:     value,
		deepMerge: false,
		matchesOn: []string{},
	}
}

func DeepMerge(value any) *MergeProp {
	return &MergeProp{
		value:     value,
		deepMerge: true,
		matchesOn: []string{},
	}
}
