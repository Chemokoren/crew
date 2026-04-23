package pagination_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/kibsoft/amy-mis/pkg/pagination"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func contextWithQuery(query string) *gin.Context {
	req := httptest.NewRequest(http.MethodGet, "/test?"+query, nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	return c
}

func TestFromContext_Defaults(t *testing.T) {
	c := contextWithQuery("")
	p := pagination.FromContext(c)

	if p.Page != 1 {
		t.Errorf("page = %d, want 1", p.Page)
	}
	if p.PerPage != 20 {
		t.Errorf("per_page = %d, want 20", p.PerPage)
	}
}

func TestFromContext_NegativePageClampedTo1(t *testing.T) {
	c := contextWithQuery("page=-5&per_page=10")
	p := pagination.FromContext(c)

	if p.Page != 1 {
		t.Errorf("page = %d, want 1 (clamped from -5)", p.Page)
	}
}

func TestFromContext_ZeroPerPageClampedToDefault(t *testing.T) {
	c := contextWithQuery("page=1&per_page=0")
	p := pagination.FromContext(c)

	if p.PerPage != 20 {
		t.Errorf("per_page = %d, want 20 (clamped from 0)", p.PerPage)
	}
}

func TestFromContext_PerPageCappedAtMax(t *testing.T) {
	c := contextWithQuery("page=1&per_page=500")
	p := pagination.FromContext(c)

	if p.PerPage != 100 {
		t.Errorf("per_page = %d, want 100 (capped from 500)", p.PerPage)
	}
}

func TestFromContext_ValidValues(t *testing.T) {
	c := contextWithQuery("page=3&per_page=50")
	p := pagination.FromContext(c)

	if p.Page != 3 {
		t.Errorf("page = %d, want 3", p.Page)
	}
	if p.PerPage != 50 {
		t.Errorf("per_page = %d, want 50", p.PerPage)
	}
}

func TestFromContext_InvalidStringsUseDefaults(t *testing.T) {
	c := contextWithQuery("page=abc&per_page=xyz")
	p := pagination.FromContext(c)

	if p.Page != 1 {
		t.Errorf("page = %d, want 1 (default for non-numeric)", p.Page)
	}
	if p.PerPage != 20 {
		t.Errorf("per_page = %d, want 20 (default for non-numeric)", p.PerPage)
	}
}

func TestParams_Offset(t *testing.T) {
	tests := []struct {
		page, perPage, wantOffset int
	}{
		{1, 20, 0},
		{2, 20, 20},
		{3, 50, 100},
		{1, 100, 0},
	}
	for _, tt := range tests {
		p := pagination.Params{Page: tt.page, PerPage: tt.perPage}
		if got := p.Offset(); got != tt.wantOffset {
			t.Errorf("Offset(page=%d, perPage=%d) = %d, want %d", tt.page, tt.perPage, got, tt.wantOffset)
		}
	}
}

func TestNewMeta(t *testing.T) {
	params := pagination.Params{Page: 2, PerPage: 10}
	meta := pagination.NewMeta(params, 25)

	if meta.Total != 25 {
		t.Errorf("total = %d, want 25", meta.Total)
	}
	if meta.TotalPages != 3 {
		t.Errorf("total_pages = %d, want 3 (ceil(25/10))", meta.TotalPages)
	}
	if meta.Page != 2 {
		t.Errorf("page = %d, want 2", meta.Page)
	}
}
