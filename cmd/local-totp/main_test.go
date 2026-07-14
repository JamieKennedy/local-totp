package main

import "testing"

func TestListenAddressDefaultsToLoopback(t *testing.T) {
	t.Setenv("LOCAL_TOTP_LISTEN_ADDR", "")

	if got := listenAddress(); got != "127.0.0.1:8080" {
		t.Fatalf("listenAddress() = %q, want loopback default", got)
	}
}

func TestListenAddressAllowsExplicitOverride(t *testing.T) {
	t.Setenv("LOCAL_TOTP_LISTEN_ADDR", ":9090")

	if got := listenAddress(); got != ":9090" {
		t.Fatalf("listenAddress() = %q, want explicit override", got)
	}
}
