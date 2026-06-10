package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go-analyze/pkg/database/entities"
)

type mockAuditDB struct {
	scrapes    []*entities.ScrapeModel
	indicators []*entities.IndicatorSnapshotModel
}

func (m *mockAuditDB) GetScrapesInRange(ctx context.Context, start, end time.Time) ([]*entities.ScrapeModel, error) {
	return m.scrapes, nil
}

func (m *mockAuditDB) GetIndicatorSnapshotsForScrapes(ctx context.Context, scrapeIDs []int) ([]*entities.IndicatorSnapshotModel, error) {
	return m.indicators, nil
}

func TestAuditHandler_RecalculatesMatchingValues(t *testing.T) {
	s := entities.NewScrape(50)
	s.SetID(1)

	// totals chosen so recompute matches reported exactly:
	// cpu_waste_cores = 1 - 0.5 = 0.5; cost = 0.5 * 0.04048 * 720 = 14.5728
	// mem_waste_GiB = (2GiB - 1GiB) = 1; cost = 1 * 0.004445 * 720 = 3.2004
	// total = 17.7732
	ind := entities.NewIndicatorSnapshot(1, 0.5, 0.5, 0, 0, 17.7732,
		1.0, 0.5,
		2*1024*1024*1024, 1*1024*1024*1024,
		0, 0, 0, 0,
	)

	db := &mockAuditDB{
		scrapes:    []*entities.ScrapeModel{s},
		indicators: []*entities.IndicatorSnapshotModel{ind},
	}
	handler := NewAuditHandler(db, 0.04048, 0.004445, 720, "fargate-test", newDiscardLogger())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit?period=1hr", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var resp AuditResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !resp.WithinTolerance {
		t.Errorf("expected within tolerance, got %f%%", resp.ErrorPercent)
	}
	if resp.CostModel != "fargate-test" {
		t.Errorf("cost model not propagated: %s", resp.CostModel)
	}
	if resp.ScrapesCount != 1 {
		t.Errorf("expected 1 scrape, got %d", resp.ScrapesCount)
	}
}

func TestAuditHandler_DetectsDivergence(t *testing.T) {
	s := entities.NewScrape(50)
	s.SetID(1)

	// reported 100, recalculation will be derived from inputs and be far from 100
	ind := entities.NewIndicatorSnapshot(1, 0.5, 0.5, 0, 0, 100.0,
		1.0, 0.5, 2*1024*1024*1024, 1*1024*1024*1024,
		0, 0, 0, 0,
	)

	db := &mockAuditDB{
		scrapes:    []*entities.ScrapeModel{s},
		indicators: []*entities.IndicatorSnapshotModel{ind},
	}
	handler := NewAuditHandler(db, 0.04048, 0.004445, 720, "fargate-test", newDiscardLogger())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit?period=1hr&details=true", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	var resp AuditResponse
	_ = json.NewDecoder(rec.Body).Decode(&resp)
	if resp.WithinTolerance {
		t.Errorf("expected divergence > 10%%, got %f%%", resp.ErrorPercent)
	}
	if len(resp.Points) != 1 {
		t.Errorf("expected 1 point, got %d", len(resp.Points))
	}
}
