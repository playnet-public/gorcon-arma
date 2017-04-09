package bercon

import "testing"

func Test_getSequence(t *testing.T) {
	var tests = []struct {
		test     []byte
		expected byte
	}{
		{
			test:     []byte{'B', 'E', 1, 1, 1, 1, 0, 1, 255, 10, 5, 2, 82},
			expected: 255,
		},
		{
			test:     []byte{'B', 'E', 1, 1, 1, 1, 0, 1, 85},
			expected: 85,
		},
	}

	for _, v := range tests {
		result, err := getSequence(v.test)
		if err != nil {
			t.Error("Packet Size mismatch")
		}
		if result != v.expected {
			t.Error("Expected:", v.expected, "Got:", result)
		}
	}
}

func Test_responseType(t *testing.T) {
	var tests = []struct {
		test     []byte
		expected byte
	}{
		{
			test:     []byte{'B', 'E', 1, 1, 1, 1, 0, 1, 255, 10, 5, 2, 82},
			expected: 1,
		},
		{
			test:     []byte{'B', 'E', 1, 1, 1, 1, 0, 1, 85},
			expected: 1,
		},
	}
	for _, v := range tests {
		result, err := responseType(v.test)
		if err != nil {
			t.Error("Test:", v.test, "Failed due to error:", err)
		}
		if result != v.expected {
			t.Error("Expected:", v.expected, "Got:", result)
		}
	}
}
