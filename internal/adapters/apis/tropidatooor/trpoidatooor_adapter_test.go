package tropidatooor

import (
	"context"
	"testing"
)

func TestTropidatooorAdapter_DataRequestAndDataRelease(t *testing.T) {
	adapter := NewTropidatooorAdapter("http://localhost:3000")
	ctx := context.Background()

	// Test DataRequest
	mount, err := adapter.DataRequest(ctx, "test-backend")
	if err != nil {
		t.Fatalf("DataRequest failed: %v", err)
	}
	t.Logf("DataRequest succeeded: %+v", mount)

	// Extract unique ID and test DataRelease
	uniqueID := mount.Id
	if err := adapter.DataRelease(ctx, uniqueID); err != nil {
		t.Fatalf("DataRelease failed: %v", err)
	}
	t.Logf("DataRelease succeeded for uniqueID: %s", uniqueID)
}
