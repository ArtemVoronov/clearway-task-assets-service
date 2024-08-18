package v1

import (
	"testing"
)

func TestParseBoundaryString(t *testing.T) {
	contentTypeString := "multipart/form-data; boundary=--------------------------086623207306110249839573"
	expectedBoundaryString := "--------------------------086623207306110249839573"

	actualBoundaryString, err := parseBoundaryString(contentTypeString)
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	if actualBoundaryString != expectedBoundaryString {
		t.Errorf("expected boundary string: %v", expectedBoundaryString)
	}
}
