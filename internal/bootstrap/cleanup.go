// Code generated from: xkit-template.
// generated at        2026-05-25 16:32:33 CST.

package bootstrap

type CleanupStack struct {
	funcs []func()
}

func (s *CleanupStack) Add(cleanup func()) {
	if cleanup == nil {
		return
	}
	s.funcs = append(s.funcs, cleanup)
}

func (s *CleanupStack) Run() {
	for i := len(s.funcs) - 1; i >= 0; i-- {
		s.funcs[i]()
	}
}
