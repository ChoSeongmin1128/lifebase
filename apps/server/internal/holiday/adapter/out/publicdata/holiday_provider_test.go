package publicdata

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseHolidayItems_Array(t *testing.T) {
	raw := json.RawMessage(`[
	  {"dateKind":"01","dateName":"삼일절","isHoliday":"Y","locdate":20260301},
	  {"dateKind":"01","dateName":"평일","isHoliday":"N","locdate":20260303}
	]`)
	items, err := parseHolidayItems(raw, 2026, 3)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("unexpected item count: %d", len(items))
	}
	if items[0].Name != "삼일절" {
		t.Fatalf("unexpected holiday name: %s", items[0].Name)
	}
}

func TestParseHolidayItems_SingleObject(t *testing.T) {
	raw := json.RawMessage(`{"dateKind":"01","dateName":"광복절","isHoliday":"Y","locdate":"20260815"}`)
	items, err := parseHolidayItems(raw, 2026, 8)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("unexpected item count: %d", len(items))
	}
	if items[0].Name != "광복절" {
		t.Fatalf("unexpected holiday name: %s", items[0].Name)
	}
}

func TestParseHolidayItems_EmptyString(t *testing.T) {
	raw := json.RawMessage(`""`)
	items, err := parseHolidayItems(raw, 2026, 8)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("unexpected item count: %d", len(items))
	}
}

func TestExtractItemRaw_EmptyItemsString(t *testing.T) {
	raw := json.RawMessage(`""`)
	itemRaw, err := extractItemRaw(raw)
	if err != nil {
		t.Fatalf("extract error: %v", err)
	}
	if len(itemRaw) != 0 {
		t.Fatalf("expected empty item raw, got: %s", string(itemRaw))
	}
}

func TestExtractItemRaw_WrappedEmptyItem(t *testing.T) {
	raw := json.RawMessage(`{"item":""}`)
	itemRaw, err := extractItemRaw(raw)
	if err != nil {
		t.Fatalf("extract error: %v", err)
	}
	if len(itemRaw) != 0 {
		t.Fatalf("expected empty item raw, got: %s", string(itemRaw))
	}
}

func TestHolidayProviderFetchMonthAndHelpers(t *testing.T) {
	if p := NewHolidayProvider(" key ", ""); p.endpoint != defaultEndpoint || p.serviceKey != "key" {
		t.Fatalf("unexpected defaults: %#v", p)
	}

	p := NewHolidayProvider("", "https://example.com")
	if _, _, err := p.FetchMonth(context.Background(), 2026, 3); err == nil {
		t.Fatal("expected missing service key error")
	}

	p = NewHolidayProvider("key", "://bad-url")
	if _, _, err := p.FetchMonth(context.Background(), 2026, 3); err == nil {
		t.Fatal("expected bad endpoint parse error")
	}

	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		switch calls {
		case 1:
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"response":{"header":{"resultCode":"99","resultMsg":"fail"}}}`))
		default:
			_, _ = w.Write([]byte(`{
				"response":{
					"header":{"resultCode":"00","resultMsg":"OK"},
					"body":{"items":{"item":[{"dateKind":"01","dateName":"삼일절","isHoliday":"Y","locdate":"20260301"}]}}
				}
			}`))
		}
	}))
	defer srv.Close()

	p = NewHolidayProvider("key", srv.URL)
	items, code, err := p.FetchMonth(context.Background(), 2026, 3)
	if err != nil {
		t.Fatalf("fetch month failed: %v", err)
	}
	if code != "00" || len(items) != 1 {
		t.Fatalf("unexpected fetch result: code=%s items=%d", code, len(items))
	}
	if calls != 2 {
		t.Fatalf("expected retry then success, calls=%d", calls)
	}
}

func TestHolidayProviderFetchOnceErrorBranches(t *testing.T) {
	ctx := context.Background()
	p := NewHolidayProvider("key", "https://example.com")

	_, _, err := p.fetchOnce(ctx, "://bad-url", 2026, 3)
	if err == nil {
		t.Fatal("expected bad request build error")
	}

	srvStatus := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srvStatus.Close()
	if _, _, err := p.fetchOnce(ctx, srvStatus.URL, 2026, 3); err == nil {
		t.Fatal("expected non-200 error")
	}

	srvDecode := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{`))
	}))
	defer srvDecode.Close()
	if _, _, err := p.fetchOnce(ctx, srvDecode.URL, 2026, 3); err == nil {
		t.Fatal("expected decode error")
	}

	srvAPIErr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"response":{"header":{"resultCode":"99","resultMsg":"invalid"}}}`))
	}))
	defer srvAPIErr.Close()
	if _, code, err := p.fetchOnce(ctx, srvAPIErr.URL, 2026, 3); err == nil || code != "99" {
		t.Fatalf("expected api error result code, code=%s err=%v", code, err)
	}

	srvItemsErr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"response":{
				"header":{"resultCode":"00","resultMsg":"OK"},
				"body":{"items":123}
			}
		}`))
	}))
	defer srvItemsErr.Close()
	if _, _, err := p.fetchOnce(ctx, srvItemsErr.URL, 2026, 3); err == nil {
		t.Fatal("expected extract item raw error")
	}
}

func TestHolidayProviderUtilityBranches(t *testing.T) {
	raw, err := extractItemRaw(json.RawMessage(`[{"dateKind":"01","dateName":"독립기념일","isHoliday":"Y","locdate":"20260301"}]`))
	if err != nil || string(raw) == "" {
		t.Fatalf("expected passthrough raw json object, err=%v raw=%s", err, string(raw))
	}
	raw, err = extractItemRaw(json.RawMessage(`"not-json"`))
	if err != nil || string(raw) != "not-json" {
		t.Fatalf("expected passthrough string payload, err=%v raw=%s", err, string(raw))
	}

	if got := normalizeLocdate(" 20260301 "); got != "20260301" {
		t.Fatalf("unexpected normalize string: %q", got)
	}
	if got := normalizeLocdate(float64(20260301)); got != "20260301" {
		t.Fatalf("unexpected normalize float: %q", got)
	}
	if got := normalizeLocdate(int64(20260301)); got != "20260301" {
		t.Fatalf("unexpected normalize int64: %q", got)
	}
	if got := normalizeLocdate(int(20260301)); got != "20260301" {
		t.Fatalf("unexpected normalize int: %q", got)
	}
	if got := normalizeLocdate(struct{ V int }{V: 1}); got == "" {
		t.Fatal("unexpected empty normalize fallback")
	}

	if isRetryable(nil) {
		t.Fatal("nil error must not be retryable")
	}
	if isRetryable(context.DeadlineExceeded) {
		t.Fatal("deadline exceeded should not be retryable with current matcher")
	}
	if !isRetryable(http.ErrHandlerTimeout) {
		t.Fatal("timeout should be retryable")
	}
	if !isRetryable(errString("kasi api returned 500")) {
		t.Fatal("5xx message should be retryable")
	}
	if !isRetryable(errString("connection reset by peer")) {
		t.Fatal("connection message should be retryable")
	}
	if isRetryable(errString("bad request")) {
		t.Fatal("bad request should not be retryable")
	}
}

func TestHolidayProviderFetchMonthNonRetryableStopsImmediately(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	p := NewHolidayProvider("key", srv.URL)
	if _, _, err := p.FetchMonth(context.Background(), 2026, 4); err == nil {
		t.Fatal("expected fetch month error")
	}
	if calls != 1 {
		t.Fatalf("expected non-retryable status to stop immediately, calls=%d", calls)
	}
}

func TestHolidayProviderFetchOnceDoError(t *testing.T) {
	p := NewHolidayProvider("key", "https://example.com")
	p.client = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("connection reset by peer")
	})}

	if _, _, err := p.fetchOnce(context.Background(), "https://example.com", 2026, 5); err == nil {
		t.Fatal("expected transport error")
	}
}

func TestHolidayProviderParseAndExtractErrorBranches(t *testing.T) {
	if _, err := extractItemRaw(json.RawMessage(`123`)); err == nil {
		t.Fatal("expected unexpected items payload error")
	}

	if _, err := parseHolidayItems(json.RawMessage(`{`), 2026, 3); err == nil {
		t.Fatal("expected object unmarshal error")
	}
	if _, err := parseHolidayItems(json.RawMessage(`[{"isHoliday":"Y"}`), 2026, 3); err == nil {
		t.Fatal("expected array unmarshal error")
	}

	items, err := parseHolidayItems(json.RawMessage(`[
	  {"dateKind":"01","dateName":"bad-locdate","isHoliday":"Y","locdate":"20263"},
	  {"dateKind":"01","dateName":"bad-date","isHoliday":"Y","locdate":"20260231"},
	  {"dateKind":"01","dateName":"valid","isHoliday":"Y","locdate":"20260301"}
	]`), 2026, 3)
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if len(items) != 1 || items[0].Name != "valid" {
		t.Fatalf("expected only valid holiday item, got %#v", items)
	}
}

func TestHolidayProviderFetchOnceUnknownResultCodeAndNullItems(t *testing.T) {
	p := NewHolidayProvider("key", "https://example.com")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"response":{
				"header":{"resultCode":"","resultMsg":"OK"},
				"body":{"items":{"item":null}}
			}
		}`))
	}))
	defer srv.Close()

	if _, code, err := p.fetchOnce(context.Background(), srv.URL, 2026, 3); err == nil || code != "UNKNOWN" {
		t.Fatalf("expected UNKNOWN result code error, code=%s err=%v", code, err)
	}
}

func TestExtractItemRawWrappedNullAndObject(t *testing.T) {
	raw, err := extractItemRaw(json.RawMessage(`{"item":null}`))
	if err != nil {
		t.Fatalf("extract wrapped null: %v", err)
	}
	if raw != nil {
		t.Fatalf("expected nil raw for wrapped null, got %s", string(raw))
	}

	raw, err = extractItemRaw(json.RawMessage(`{"item":{"dateKind":"01"}}`))
	if err != nil {
		t.Fatalf("extract wrapped object: %v", err)
	}
	if string(raw) == "" {
		t.Fatal("expected non-empty wrapped object raw")
	}
}

func TestExtractItemRawEmptyObjectVariants(t *testing.T) {
	cases := []json.RawMessage{
		json.RawMessage(`{}`),
		json.RawMessage(`null`),
		json.RawMessage(`{"item":{}}`),
	}

	for _, raw := range cases {
		itemRaw, err := extractItemRaw(raw)
		if err != nil {
			t.Fatalf("extract item raw for %s: %v", string(raw), err)
		}
		if itemRaw != nil {
			t.Fatalf("expected nil item raw for %s, got %s", string(raw), string(itemRaw))
		}
	}
}

func TestExtractItemRawMalformedJSONString(t *testing.T) {
	if _, err := extractItemRaw(json.RawMessage(`"unterminated`)); err == nil {
		t.Fatal("expected malformed json string error")
	}
}

func TestExtractItemRawMalformedObjectErrors(t *testing.T) {
	if _, err := extractItemRaw(json.RawMessage(`{"dateName":`)); err == nil {
		t.Fatal("expected malformed object error")
	}
}

func TestHolidayProviderFetchOnceServerErrorAndEmptyItemsSuccess(t *testing.T) {
	p := NewHolidayProvider("key", "https://example.com")

	srv500 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv500.Close()
	if _, _, err := p.fetchOnce(context.Background(), srv500.URL, 2026, 3); err == nil {
		t.Fatal("expected 5xx fetchOnce error")
	}

	srvEmpty := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"response":{
				"header":{"resultCode":"00","resultMsg":"OK"},
				"body":{"items":null}
			}
		}`))
	}))
	defer srvEmpty.Close()
	items, code, err := p.fetchOnce(context.Background(), srvEmpty.URL, 2026, 3)
	if err != nil {
		t.Fatalf("expected empty item success, got err=%v", err)
	}
	if code != "00" || len(items) != 0 {
		t.Fatalf("expected empty success result, code=%s items=%d", code, len(items))
	}
}

func TestHolidayProviderFetchOnceParseHolidayItemsError(t *testing.T) {
	p := NewHolidayProvider("key", "https://example.com")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"response":{
				"header":{"resultCode":"00","resultMsg":"OK"},
				"body":{"items":"not-json"}
			}
		}`))
	}))
	defer srv.Close()

	if _, _, err := p.fetchOnce(context.Background(), srv.URL, 2026, 3); err == nil {
		t.Fatal("expected parseHolidayItems error from invalid string payload")
	}
}

func TestExtractItemRawPlainObjectPayload(t *testing.T) {
	raw, err := extractItemRaw(json.RawMessage(`{"dateKind":"01","dateName":"삼일절","isHoliday":"Y","locdate":"20260301"}`))
	if err != nil {
		t.Fatalf("expected plain object passthrough, got %v", err)
	}
	if string(raw) == "" {
		t.Fatal("expected non-empty plain object raw")
	}
}

func TestExtractItemRawTrimmedEmptyJSONString(t *testing.T) {
	raw, err := extractItemRaw(json.RawMessage(`"   "`))
	if err != nil {
		t.Fatalf("extract trimmed empty string: %v", err)
	}
	if raw != nil {
		t.Fatalf("expected nil raw for trimmed empty string, got %s", string(raw))
	}
}

type errString string

func (e errString) Error() string { return string(e) }

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
