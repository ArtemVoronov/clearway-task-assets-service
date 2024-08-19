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

func TestParseAuthorizationHeader(t *testing.T) {
	header := "Bearer 0a7fb23ca220aa9f1b59fef0e63b97ce"
	expected := "0a7fb23ca220aa9f1b59fef0e63b97ce"

	actual, err := parseAuthorizationHeader(header)
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	if expected != actual {
		t.Errorf("expected access token: %v", expected)
	}
}
