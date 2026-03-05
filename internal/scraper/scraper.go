package scraper

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

func Fetch(ctx context.Context, client *http.Client, url string) (int, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; MarketNewsBot/1.0)")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml")

	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return resp.StatusCode, body, fmt.Errorf("non-2xx status")
	}
	return resp.StatusCode, body, nil
}

func ParseDays(body []byte) ([]Day, error) {
	idx := bytes.Index(body, []byte("days:"))
	if idx < 0 {
		idx = bytes.Index(body, []byte(`"days":`))
	}
	if idx < 0 {
		return nil, fmt.Errorf("days field not found")
	}

	after := body[idx:]
	openIdx := bytes.IndexByte(after, '[')
	if openIdx < 0 {
		return nil, fmt.Errorf("days array start not found")
	}
	start := idx + openIdx
	end, err := findJSONEnd(body, start)
	if err != nil {
		return nil, err
	}

	raw := body[start : end+1]
	raw = sanitizeJSON(raw)

	var days []Day
	if err := json.Unmarshal(raw, &days); err != nil {
		return nil, fmt.Errorf("json unmarshal days: %w", err)
	}
	return days, nil
}

func sanitizeJSON(in []byte) []byte {
	re := regexp.MustCompile(`(?s)/\*.*?\*/`)
	out := re.ReplaceAll(in, []byte(""))
	re = regexp.MustCompile(`(?m)//.*$`)
	out = re.ReplaceAll(out, []byte(""))
	return out
}

func findJSONEnd(b []byte, start int) (int, error) {
	level := 0
	inStr := false
	escape := false
	for i := start; i < len(b); i++ {
		c := b[i]
		if inStr {
			if escape {
				escape = false
				continue
			}
			if c == '\\' {
				escape = true
				continue
			}
			if c == '"' {
				inStr = false
			}
			continue
		}
		if c == '"' {
			inStr = true
			continue
		}
		if c == '[' {
			level++
		} else if c == ']' {
			level--
			if level == 0 {
				return i, nil
			}
		}
	}
	return 0, fmt.Errorf("unterminated days array")
}

func FormatEventDatetime(ev Event) string {
	if ev.Dateline > 0 {
		return time.Unix(ev.Dateline, 0).UTC().Format(time.RFC3339)
	}
	if ev.Date != "" && ev.TimeLabel != "" {
		raw := fmt.Sprintf("%s %s", ev.Date, ev.TimeLabel)
		if t, err := time.Parse("Jan 2, 2006 3:04pm", raw); err == nil {
			return t.UTC().Format(time.RFC3339)
		}
		return raw
	}
	return ev.Date
}

func TimeLabelHasDigits(label string) bool {
	for i := 0; i < len(label); i++ {
		c := label[i]
		if c >= '0' && c <= '9' {
			return true
		}
	}
	return false
}

func BuildMeta(ev Event) map[string]interface{} {
	meta := map[string]interface{}{}
	if ev.PrefixedName != "" {
		meta["prefixedName"] = ev.PrefixedName
	}
	if ev.TrimmedPrefixed != "" {
		meta["trimmedPrefixedName"] = ev.TrimmedPrefixed
	}
	if ev.SoloTitle != "" {
		meta["soloTitle"] = ev.SoloTitle
	}
	if ev.SoloTitleFull != "" {
		meta["soloTitleFull"] = ev.SoloTitleFull
	}
	if ev.SoloTitleShort != "" {
		meta["soloTitleShort"] = ev.SoloTitleShort
	}
	if ev.Notice != "" {
		meta["notice"] = ev.Notice
	}
	if ev.URL != "" {
		meta["url"] = ev.URL
	}
	if ev.SoloURL != "" {
		meta["soloUrl"] = ev.SoloURL
	}
	if ev.ImpactTitle != "" {
		meta["impactTitle"] = ev.ImpactTitle
	}
	meta["hasNotice"] = ev.HasNotice
	meta["hasDataValues"] = ev.HasDataValues
	meta["hasGraph"] = ev.HasGraph
	meta["hasLinkedThreads"] = ev.HasLinkedThreads
	return meta
}

func DayURL(d time.Time) string {
	daySlug := strings.ToLower(d.Format("Jan2.2006"))
	return "https://www.forexfactory.com/calendar?day=" + daySlug
}
