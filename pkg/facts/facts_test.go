package facts

import "testing"

func TestCollect(t *testing.T) {
	collected, err := Collect()
	if err != nil {
		t.Fatal(err)
	}

	if collected.SystemCertificateVersion == nil {
		t.Fatal("SystemCertificateVersion is nil")
	}
}
