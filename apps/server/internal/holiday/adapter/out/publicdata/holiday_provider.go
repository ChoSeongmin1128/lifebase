package publicdata

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"lifebase/internal/holiday/domain"
	portout "lifebase/internal/holiday/port/out"
)

const (
	defaultEndpoint = "https://apis.data.go.kr/B090041/openapi/service/SpcdeInfoService/getRestDeInfo"
)

type holidayProvider struct {
	serviceKey string
	endpoint   string
	client     *http.Client
}

func NewHolidayProvider(serviceKey, endpoint string) *holidayProvider {
	if strings.TrimSpace(endpoint) == "" {
		endpoint = defaultEndpoint
	}
	return &holidayProvider{
		serviceKey: strings.TrimSpace(serviceKey),
		endpoint:   endpoint,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (p *holidayProvider) FetchMonth(ctx context.Context, year, month int) ([]domain.Holiday, string, error) {
	if strings.TrimSpace(p.serviceKey) == "" {
		return nil, "", fmt.Errorf("KASI_HOLIDAY_SERVICE_KEY is empty")
	}

	endpoint, err := url.Parse(p.endpoint)
	if err != nil {
		return nil, "", err
	}

	params := endpoint.Query()
	params.Set("serviceKey", p.serviceKey)
	params.Set("solYear", strconv.Itoa(year))
	params.Set("solMonth", fmt.Sprintf("%02d", month))
	params.Set("numOfRows", "100")
	params.Set("_type", "json")
	endpoint.RawQuery = params.Encode()

	var lastErr error
	for attempt := 0; attempt < 3; attempt += 1 {
		items, resultCode, err := p.fetchOnce(ctx, endpoint.String(), year, month)
		if err == nil {
			return items, resultCode, nil
		}
		lastErr = err
		if !isRetryable(err) || attempt == 2 {
			break
		}
		time.Sleep(time.Duration(200*(attempt+1)) * time.Millisecond)
	}

	return nil, "", lastErr
}

func (p *holidayProvider) fetchOnce(ctx context.Context, endpoint string, year, month int) ([]domain.Holiday, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, "", err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		return nil, "", fmt.Errorf("kasi api returned %d", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("kasi api returned %d", resp.StatusCode)
	}

	var payload struct {
		Response struct {
			Header struct {
				ResultCode string `json:"resultCode"`
				ResultMsg  string `json:"resultMsg"`
			} `json:"header"`
			Body struct {
				Items struct {
					Item json.RawMessage `json:"item"`
				} `json:"items"`
			} `json:"body"`
		} `json:"response"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, "", err
	}

	resultCode := payload.Response.Header.ResultCode
	if resultCode == "" {
		resultCode = "UNKNOWN"
	}
	if resultCode != "00" {
		return nil, resultCode, fmt.Errorf("kasi api error: %s", payload.Response.Header.ResultMsg)
	}

	items, err := parseHolidayItems(payload.Response.Body.Items.Item, year, month)
	if err != nil {
		return nil, resultCode, err
	}

	return items, resultCode, nil
}

func parseHolidayItems(raw json.RawMessage, year, month int) ([]domain.Holiday, error) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" || trimmed == "{}" {
		return []domain.Holiday{}, nil
	}

	type item struct {
		DateKind  string      `json:"dateKind"`
		DateName  string      `json:"dateName"`
		IsHoliday string      `json:"isHoliday"`
		Locdate   interface{} `json:"locdate"`
	}

	parsed := make([]item, 0, 4)
	if strings.HasPrefix(trimmed, "[") {
		if err := json.Unmarshal(raw, &parsed); err != nil {
			return nil, err
		}
	} else {
		var one item
		if err := json.Unmarshal(raw, &one); err != nil {
			return nil, err
		}
		parsed = append(parsed, one)
	}

	result := make([]domain.Holiday, 0, len(parsed))
	for _, it := range parsed {
		if strings.ToUpper(strings.TrimSpace(it.IsHoliday)) != "Y" {
			continue
		}

		loc := normalizeLocdate(it.Locdate)
		if len(loc) != 8 {
			continue
		}
		date, err := time.Parse("20060102", loc)
		if err != nil {
			continue
		}

		result = append(result, domain.Holiday{
			Date:      date,
			Name:      strings.TrimSpace(it.DateName),
			Year:      year,
			Month:     month,
			DateKind:  strings.TrimSpace(it.DateKind),
			IsHoliday: true,
		})
	}
	return result, nil
}

func normalizeLocdate(value interface{}) string {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case float64:
		return strconv.FormatInt(int64(v), 10)
	case int64:
		return strconv.FormatInt(v, 10)
	case int:
		return strconv.Itoa(v)
	default:
		return fmt.Sprint(v)
	}
}

func isRetryable(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "timeout") || strings.Contains(msg, "returned 5") || strings.Contains(msg, "connection")
}

var _ portout.HolidayProvider = (*holidayProvider)(nil)
