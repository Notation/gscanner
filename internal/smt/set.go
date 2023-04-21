package smt

type Set struct {
	elements map[Annotation]struct{}
}

func NewSet(elements ...Annotation) *Set {
	s := &Set{
		elements: make(map[Annotation]struct{}, len(elements)),
	}
	for _, elem := range elements {
		s.elements[elem] = struct{}{}
	}
	return s
}

func (set *Set) Union(other *Set) *Set {
	union := make(map[Annotation]struct{})
	for elem := range set.elements {
		union[elem] = struct{}{}
	}
	for elem := range other.elements {
		union[elem] = struct{}{}
	}
	return &Set{
		elements: union,
	}
}

func (set *Set) Intersect(other Set) Set {
	intersect := make(map[Annotation]struct{})
	var (
		larggerSet = set.elements
		otherSet   = other.elements
	)
	if len(set.elements) < len(other.elements) {
		larggerSet, otherSet = otherSet, larggerSet
	}
	for elem := range otherSet {
		if _, ok := larggerSet[elem]; !ok {
			continue
		}
		intersect[elem.Clone()] = struct{}{}
	}
	return Set{
		elements: intersect,
	}
}

func (set *Set) Clone() *Set {
	clone := make(map[Annotation]struct{}, len(set.elements))
	for elem := range set.elements {
		clone[elem.Clone()] = struct{}{}
	}
	return &Set{
		elements: clone,
	}
}

func (set *Set) Add(Annotation Annotation) {
	set.elements[Annotation] = struct{}{}
}

func (set *Set) GetElements() map[Annotation]struct{} {
	return set.elements
}
