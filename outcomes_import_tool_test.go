package main

import (
  "testing"
)

func TestNormalizeDomain(t *testing.T) {
  if normalizeDomain("localhost") != "http://localhost:3000" {
    t.Fatal("localhost not normalized properly")
  }
}
