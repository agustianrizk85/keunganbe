// Package gsheets fetches sheet data from a Google Spreadsheet via the Sheets
// API using a service-account credential. Values are returned as raw strings
// (UNFORMATTED_VALUE + SERIAL_NUMBER dates) so they match the engine's XLSX
// RawCellValue expectations — letting the same ingest engine process a live
// Google Sheet or an uploaded file.
package gsheets

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/oauth2/google"
)

const apiBase = "https://sheets.googleapis.com/v4/spreadsheets/"

// Client reads Google Sheets with a service-account credential.
type Client struct {
	cred []byte
}

// New loads the service-account JSON at credPath. An empty path returns
// (nil, nil) — the sync feature is then disabled with a clear API error.
func New(credPath string) (*Client, error) {
	if strings.TrimSpace(credPath) == "" {
		return nil, nil
	}
	b, err := os.ReadFile(credPath)
	if err != nil {
		return nil, fmt.Errorf("baca kredensial Google: %w", err)
	}
	return &Client{cred: b}, nil
}

// FetchAll returns every tab of the spreadsheet as raw string rows, keyed by the
// tab's actual title. The finance ingest engine classifies sheets by their
// header signature, so it does not need fixed tab names.
func (c *Client) FetchAll(ctx context.Context, spreadsheetID string) (map[string][][]string, error) {
	httpClient, err := c.client(ctx)
	if err != nil {
		return nil, err
	}
	titles, err := fetchTitles(httpClient, spreadsheetID)
	if err != nil {
		return nil, err
	}
	if len(titles) == 0 {
		return nil, fmt.Errorf("spreadsheet tidak punya tab")
	}
	u := apiBase + url.PathEscape(spreadsheetID) +
		"/values:batchGet?valueRenderOption=UNFORMATTED_VALUE&dateTimeRenderOption=SERIAL_NUMBER"
	for _, t := range titles {
		u += "&ranges=" + url.QueryEscape("'"+t+"'")
	}
	var resp struct {
		ValueRanges []struct {
			Values [][]interface{} `json:"values"`
		} `json:"valueRanges"`
	}
	if err := getJSON(httpClient, u, &resp); err != nil {
		return nil, err
	}
	out := make(map[string][][]string, len(titles))
	for i, t := range titles {
		if i >= len(resp.ValueRanges) {
			break
		}
		out[t] = toStrings(resp.ValueRanges[i].Values)
	}
	return out, nil
}

func (c *Client) client(ctx context.Context) (*http.Client, error) {
	conf, err := google.JWTConfigFromJSON(c.cred, "https://www.googleapis.com/auth/spreadsheets.readonly")
	if err != nil {
		return nil, fmt.Errorf("kredensial Google tidak valid: %w", err)
	}
	hc := conf.Client(ctx)
	hc.Timeout = 90 * time.Second
	return hc, nil
}

func fetchTitles(client *http.Client, id string) ([]string, error) {
	var meta struct {
		Sheets []struct {
			Properties struct {
				Title string `json:"title"`
			} `json:"properties"`
		} `json:"sheets"`
	}
	u := apiBase + url.PathEscape(id) + "?fields=sheets.properties.title"
	if err := getJSON(client, u, &meta); err != nil {
		return nil, err
	}
	titles := make([]string, 0, len(meta.Sheets))
	for _, s := range meta.Sheets {
		titles = append(titles, s.Properties.Title)
	}
	return titles, nil
}

func getJSON(client *http.Client, u string, dst interface{}) error {
	res, err := client.Get(u)
	if err != nil {
		return fmt.Errorf("akses Google Sheets gagal: %w", err)
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(res.Body, 200<<20))
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("Google Sheets HTTP %d: %s", res.StatusCode, shorten(string(body)))
	}
	return json.Unmarshal(body, dst)
}

func toStrings(rows [][]interface{}) [][]string {
	out := make([][]string, len(rows))
	for i, r := range rows {
		row := make([]string, len(r))
		for j, v := range r {
			row[j] = cellToString(v)
		}
		out[i] = row
	}
	return out
}

// cellToString mirrors XLSX RawCellValue: numbers/dates as plain decimals (no
// scientific notation, so long phone numbers stay intact), strings as-is.
func cellToString(v interface{}) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case bool:
		if x {
			return "TRUE"
		}
		return "FALSE"
	case float64:
		if x == math.Trunc(x) && math.Abs(x) < 1e15 {
			return strconv.FormatInt(int64(x), 10)
		}
		return strconv.FormatFloat(x, 'f', -1, 64)
	case json.Number:
		return x.String()
	default:
		return fmt.Sprintf("%v", x)
	}
}

func shorten(s string) string {
	if len(s) > 300 {
		return s[:300] + "…"
	}
	return s
}
