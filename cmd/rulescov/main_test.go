package main

import "testing"

func TestExtrairIDs(t *testing.T) {
	ids := extrairIDs("Casos A01, SR12, T07, R9, E2, EX1, PADS-E05, D1, P1, V2 e S3; conforme V3 e V4; R$ 10; AB12; T123.")
	for _, id := range []string{"A01", "SR12", "T07", "R9", "E2", "EX1", "PADS-E05", "D1", "P1", "V2", "S3"} {
		if !ids[id] {
			t.Errorf("faltou %s", id)
		}
	}
	for _, id := range []string{"V3", "V4", "AB12", "T123", "R"} {
		if ids[id] {
			t.Errorf("falso positivo %s", id)
		}
	}
}
