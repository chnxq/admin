package service

import "testing"

type permissionTestingWriter struct {
	t *testing.T
}

func (w permissionTestingWriter) Write(p []byte) (int, error) {
	if w.t != nil {
		w.t.Log(string(p))
	}
	return len(p), nil
}
