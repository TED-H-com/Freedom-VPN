package fakecontrol

import "testing"

func TestStartAndClose(t *testing.T) {
	s, err := Start()
	if err != nil {
		t.Fatal(err)
	}
	if s.URL == "" || len(s.TrustJSON()) == 0 || len(s.CACertPEM()) == 0 {
		t.Fatal("server did not publish trust material")
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
}
