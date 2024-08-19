package v1

import "testing"

func TestParseIpAddr(t *testing.T) {
	remoteAddr := "192.168.65.1:62297"
	expectedIpAddr := "192.168.65.1"

	actualIpAddr, err := parseIpAddr(remoteAddr)
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	if actualIpAddr != expectedIpAddr {
		t.Errorf("expected ip address: %v", expectedIpAddr)
	}
}
