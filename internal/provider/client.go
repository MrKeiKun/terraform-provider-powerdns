package provider

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	freecache "github.com/coocood/freecache"
	cleanhttp "github.com/hashicorp/go-cleanhttp"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// DefaultSchema is the value used for the URL in case
// no schema is explicitly defined.
var DefaultSchema = "https"

// DefaultCacheSize is client default cache size.
var DefaultCacheSize int

// Client is a PowerDNS client representation.
type Client struct {
	ServerURL         string // Location of PowerDNS authoritative server to use
	RecursorServerURL string // Location of PowerDNS recursor server to use
	ServerVersion     string
	APIKey            string // REST API Static authentication key
	APIVersion        int    // API version to use
	HTTP              *http.Client
	CacheEnable       bool // Enable/Disable cache for REST API requests
	Cache             *freecache.Cache
	CacheTTL          int
}

// NewClient returns a new PowerDNS client.
func NewClient(ctx context.Context, serverURL string, recursorServerURL string, apiKey string, configTLS *tls.Config, cacheEnable bool, cacheSizeMB string, cacheTTL int) (*Client, error) {
	// Input validation
	if serverURL == "" {
		return nil, fmt.Errorf("serverURL cannot be empty")
	}
	if recursorServerURL == "" {
		return nil, fmt.Errorf("recursorServerURL cannot be empty")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("apiKey cannot be empty")
	}
	if cacheTTL < 0 {
		return nil, fmt.Errorf("cacheTTL cannot be negative")
	}

	// Sanitize URLs
	cleanURL, err := sanitizeURL(serverURL)
	if err != nil {
		return nil, fmt.Errorf("failed to sanitize server URL: %w", err)
	}

	cleanRecursorURL, err := sanitizeURL(recursorServerURL)
	if err != nil {
		return nil, fmt.Errorf("failed to sanitize recursor server URL: %w", err)
	}

	// Setup HTTP client
	httpClient := cleanhttp.DefaultClient()
	if transport, ok := httpClient.Transport.(*http.Transport); ok && configTLS != nil {
		transport.TLSClientConfig = configTLS
	}

	// Initialize cache if enabled
	var cache *freecache.Cache
	if cacheEnable {
		cacheSize, err := parseCacheSizeMB(cacheSizeMB)
		if err != nil {
			return nil, fmt.Errorf("failed to parse cache size: %w", err)
		}
		cache = freecache.NewCache(cacheSize)
	}

	// Create client
	client := &Client{
		ServerURL:         cleanURL,
		RecursorServerURL: cleanRecursorURL,
		APIKey:            apiKey,
		HTTP:              httpClient,
		APIVersion:        -1,
		CacheEnable:       cacheEnable,
		Cache:             cache,
		CacheTTL:          cacheTTL,
	}

	// Set server version (optional)
	if err := client.setServerVersion(ctx); err != nil {
		tflog.Warn(ctx, "Failed to set server version, continuing without it", map[string]interface{}{
			"error": err.Error(),
		})
	}

	return client, nil
}

// parseCacheSizeMB parses cache size in MB and returns bytes.
func parseCacheSizeMB(cacheSizeMB string) (int, error) {
	size, err := strconv.Atoi(cacheSizeMB)
	if err != nil {
		return 0, fmt.Errorf("invalid cache size: %w", err)
	}
	if size <= 0 {
		return 0, fmt.Errorf("cache size must be positive")
	}
	return size * 1024 * 1024, nil
}

// sanitizeURL will output:
// <scheme>://<host>[:port]
// with no trailing /.
func sanitizeURL(URL string) (string, error) {
	cleanURL := ""
	host := ""
	schema := ""

	var err error

	if len(URL) == 0 {
		return "", fmt.Errorf("no URL provided")
	}

	// Trim surrounding quotes that may be included from environment variables
	URL = strings.Trim(URL, "\"'")

	parsedURL, err := url.Parse(URL)
	if err != nil {
		return "", fmt.Errorf("error while trying to parse URL: %s", err)
	}

	if len(parsedURL.Scheme) == 0 {
		schema = DefaultSchema
	} else {
		if (parsedURL.Scheme == "http") || (parsedURL.Scheme == "https") {
			schema = parsedURL.Scheme
		} else {
			schema = DefaultSchema
		}
	}

	if len(parsedURL.Host) == 0 {
		tryout, _ := url.Parse(schema + "://" + URL)

		if len(tryout.Host) == 0 {
			return "", fmt.Errorf("unable to find a hostname in '%s'", URL)
		}

		host = tryout.Host
	} else {
		host = parsedURL.Host
	}

	cleanURL = schema + "://" + host

	return cleanURL, nil
}

// Creates a new request with necessary headers.
func (client *Client) newRequest(ctx context.Context, method string, endpoint string, body []byte) (*http.Request, error) {
	var err error
	if client.APIVersion < 0 {
		client.APIVersion, err = client.detectAPIVersion(ctx)
	}
	if err != nil {
		return nil, err
	}

	var urlStr string
	if client.APIVersion > 0 {
		urlStr = client.ServerURL + "/api/v" + strconv.Itoa(client.APIVersion) + endpoint
	} else {
		urlStr = client.ServerURL + endpoint
	}
	u, err := url.Parse(urlStr)
	if err != nil {
		return nil, fmt.Errorf("error during parsing request URL: %s", err)
	}

	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequest(method, u.String(), bodyReader)
	if err != nil {
		return nil, fmt.Errorf("error during creation of request: %s", err)
	}

	req.Header.Add("X-API-Key", client.APIKey)
	req.Header.Add("Accept", "application/json")

	if method != http.MethodGet {
		req.Header.Add("Content-Type", "application/json")
	}

	return req, nil
}

// Creates a new request for recursor API.
func (client *Client) newRequestRecursor(ctx context.Context, method string, endpoint string, body []byte) (*http.Request, error) {
	var urlStr string
	urlStr = client.RecursorServerURL + "/api/v1" + endpoint

	u, err := url.Parse(urlStr)
	if err != nil {
		return nil, fmt.Errorf("error during parsing request URL: %s", err)
	}

	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequest(method, u.String(), bodyReader)
	if err != nil {
		return nil, fmt.Errorf("error during creation of request: %s", err)
	}

	req.Header.Add("X-API-Key", client.APIKey)
	req.Header.Add("Accept", "application/json")

	if method != http.MethodGet {
		req.Header.Add("Content-Type", "application/json")
	}

	return req, nil
}

// RecursorZone represents a PowerDNS recursor zone object.
type RecursorZone struct {
	ID               string              `json:"id"`
	Name             string              `json:"name"`
	Type             string              `json:"type"`
	Kind             string              `json:"kind"`
	Servers          []string            `json:"servers"`
	RecursionDesired bool                `json:"recursion_desired"`
	NotifyAllowed    bool                `json:"notify_allowed"`
	URL              string              `json:"url"`
	RRSets           []ResourceRecordSet `json:"rrsets"`
}

// ZoneInfo represents a PowerDNS zone object.
type ZoneInfo struct {
	ID                 string              `json:"id"`
	Name               string              `json:"name"`
	URL                string              `json:"url"`
	Kind               string              `json:"kind"`
	DNSSec             bool                `json:"dnsssec"`
	Serial             int64               `json:"serial"`
	Records            []Record            `json:"records,omitempty"`
	ResourceRecordSets []ResourceRecordSet `json:"rrsets,omitempty"`
	Account            string              `json:"account"`
	Nameservers        []string            `json:"nameservers,omitempty"`
	Masters            []string            `json:"masters,omitempty"`
	SoaEditAPI         string              `json:"soa_edit_api"`
}

// ZoneInfoUpd is a limited subset for supported updates.
type ZoneInfoUpd struct {
	Name       string `json:"name"`
	Kind       string `json:"kind"`
	SoaEditAPI string `json:"soa_edit_api,omitempty"`
	Account    string `json:"account"`
}

// Record represents a PowerDNS record object.
type Record struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Content  string `json:"content"`
	TTL      int    `json:"ttl"` // For API v0
	Disabled bool   `json:"disabled"`
	SetPtr   bool   `json:"set-ptr"`
}

// ResourceRecordSet represents a PowerDNS RRSet object.
type ResourceRecordSet struct {
	Name       string   `json:"name"`
	Type       string   `json:"type"`
	ChangeType string   `json:"changetype"`
	TTL        int      `json:"ttl"` // For API v1
	Records    []Record `json:"records,omitempty"`
}

type zonePatchRequest struct {
	RecordSets []ResourceRecordSet `json:"rrsets"`
}

type errorResponse struct {
	ErrorMsg string `json:"error"`
}

type serverInfo struct {
	ConfigURL  string `json:"config_url"`
	DaemonType string `json:"daemon_type"`
	ID         string `json:"id"`
	Type       string `json:"type"`
	URL        string `json:"url"`
	Version    string `json:"version"`
	ZonesURL   string `json:"zones_url"`
}

const idSeparator string = ":::"

// Sentinel error for "not found" scenarios.
var (
	// ErrNotFound is returned when a resource is not found.
	ErrNotFound = errors.New("not found")
)

// ID returns a record with the ID format.
func (record *Record) ID() string {
	return record.Name + idSeparator + record.Type
}

// ID returns a rrSet with the ID format.
func (rrSet *ResourceRecordSet) ID() string {
	return rrSet.Name + idSeparator + rrSet.Type
}

// Returns name and type of record or record set based on its ID.
func parseID(recID string) (string, string, error) {
	s := strings.Split(recID, idSeparator)
	if len(s) == 2 {
		return s[0], s[1], nil
	}
	return "", "", fmt.Errorf("unknown record ID format")
}

// Detects the API version in use on the server
// Uses int to represent the API version: 0 is the legacy AKA version 3.4 API
// Any other integer correlates with the same API version.
func (client *Client) detectAPIVersion(ctx context.Context) (int, error) {
	httpClient := client.HTTP

	u, err := url.Parse(client.ServerURL + "/api/v1/servers")
	if err != nil {
		return -1, fmt.Errorf("error while trying to detect the API version, request URL: %s", err)
	}

	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return -1, fmt.Errorf("error during creation of request: %s", err)
	}

	req.Header.Add("X-API-Key", client.APIKey)
	req.Header.Add("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return -1, err
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			tflog.Warn(ctx, "Error closing response body", map[string]interface{}{
				"error":  err.Error(),
				"method": req.Method,
				"url":    req.URL.String(),
			})
		}
	}()

	if resp.StatusCode == http.StatusOK {
		return 1, nil
	}
	return 0, nil
}

// ListZones returns all Zones of server, without records.
func (client *Client) ListZones(ctx context.Context) ([]ZoneInfo, error) {
	var zoneInfos []ZoneInfo
	err := client.doRequest(ctx, http.MethodGet, "/servers/localhost/zones", nil, http.StatusOK, &zoneInfos)
	return zoneInfos, err
}

// GetZone gets a zone.
func (client *Client) GetZone(ctx context.Context, name string) (ZoneInfo, error) {
	var zoneInfo ZoneInfo
	err := client.doRequest(ctx, http.MethodGet, fmt.Sprintf("/servers/localhost/zones/%s", name), nil, http.StatusOK, &zoneInfo)
	return zoneInfo, err
}

// ZoneExists checks if requested zone exists.
func (client *Client) ZoneExists(ctx context.Context, name string) (bool, error) {
	req, err := client.newRequest(ctx, http.MethodGet, fmt.Sprintf("/servers/localhost/zones/%s", name), nil)
	if err != nil {
		return false, err
	}

	resp, err := client.HTTP.Do(req)
	if err != nil {
		return false, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			tflog.Warn(ctx, "Error closing response body", map[string]interface{}{
				"error":  err.Error(),
				"method": req.Method,
				"url":    req.URL.String(),
				"zone":   name,
			})
		}
	}()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		errorResp := new(errorResponse)
		if err = json.NewDecoder(resp.Body).Decode(errorResp); err != nil {
			return false, fmt.Errorf("error getting zone: %s", name)
		}
		return false, fmt.Errorf("error getting zone: %s, reason: %q", name, errorResp.ErrorMsg)
	}

	return resp.StatusCode == http.StatusOK, nil
}

// CreateZone creates a zone.
func (client *Client) CreateZone(ctx context.Context, zoneInfo ZoneInfo) (ZoneInfo, error) {
	body, err := json.Marshal(zoneInfo)
	if err != nil {
		return ZoneInfo{}, err
	}

	var createdZoneInfo ZoneInfo
	err = client.doRequest(ctx, http.MethodPost, "/servers/localhost/zones", body, http.StatusCreated, &createdZoneInfo)
	return createdZoneInfo, err
}

// UpdateZone updates a zone.
func (client *Client) UpdateZone(ctx context.Context, name string, zoneInfo ZoneInfoUpd) error {
	body, err := json.Marshal(zoneInfo)
	if err != nil {
		return err
	}

	return client.doRequest(ctx, http.MethodPut, fmt.Sprintf("/servers/localhost/zones/%s", name), body, http.StatusNoContent, nil)
}

// DeleteZone deletes a zone.
func (client *Client) DeleteZone(ctx context.Context, name string) error {
	return client.doRequest(ctx, http.MethodDelete, fmt.Sprintf("/servers/localhost/zones/%s", name), nil, http.StatusNoContent, nil)
}

// GetZoneInfoFromCache return ZoneInfo struct.
func (client *Client) GetZoneInfoFromCache(ctx context.Context, zone string) (*ZoneInfo, error) {
	if client.CacheEnable {
		cacheZoneInfo, err := client.Cache.Get([]byte(zone))
		if err != nil {
			return nil, err
		}

		zoneInfo := new(ZoneInfo)
		if err := json.Unmarshal(cacheZoneInfo, &zoneInfo); err != nil {
			return nil, err
		}

		return zoneInfo, nil
	}

	return nil, nil
}

// ListRecords returns all records in Zone.
func (client *Client) ListRecords(ctx context.Context, zone string) ([]Record, error) {
	zoneInfo, err := client.GetZoneInfoFromCache(ctx, zone)
	if err != nil {
		tflog.Warn(ctx, "Cache get failed", map[string]interface{}{
			"zone":  zone,
			"error": err.Error(),
		})
		return nil, err
	}

	if zoneInfo == nil {
		req, err := client.newRequest(ctx, http.MethodGet, fmt.Sprintf("/servers/localhost/zones/%s", zone), nil)
		if err != nil {
			return nil, err
		}

		resp, err := client.HTTP.Do(req)
		if err != nil {
			return nil, err
		}
		defer func() {
			if err := resp.Body.Close(); err != nil {
				tflog.Warn(ctx, "Error closing response body", map[string]interface{}{
					"error":  err.Error(),
					"method": req.Method,
					"url":    req.URL.String(),
					"zone":   zone,
				})
			}
		}()

		zoneInfo = new(ZoneInfo)
		if err := json.NewDecoder(resp.Body).Decode(zoneInfo); err != nil {
			return nil, err
		}

		if client.CacheEnable {
			cacheValue, err := json.Marshal(zoneInfo)
			if err != nil {
				return nil, err
			}

			if err := client.Cache.Set([]byte(zone), cacheValue, client.CacheTTL); err != nil {
				return nil, fmt.Errorf("the cache for REST API requests is enabled but the size isn't enough: cacheSize: %db \n %s",
					DefaultCacheSize, err)
			}
		}
	}

	records := zoneInfo.Records
	// Convert the API v1 response to v0 record structure
	for _, rrs := range zoneInfo.ResourceRecordSets {
		for _, record := range rrs.Records {
			records = append(records, Record{
				Name:    rrs.Name,
				Type:    rrs.Type,
				Content: record.Content,
				TTL:     rrs.TTL,
			})
		}
	}

	return records, nil
}

// ListRecordsInRRSet returns only records of specified name and type.
func (client *Client) ListRecordsInRRSet(ctx context.Context, zone string, name string, tpe string) ([]Record, error) {
	allRecords, err := client.ListRecords(ctx, zone)
	if err != nil {
		return nil, err
	}

	records := make([]Record, 0, 10)
	for _, r := range allRecords {
		if strings.EqualFold(r.Name, name) && strings.EqualFold(r.Type, tpe) {
			records = append(records, r)
		}
	}

	return records, nil
}

// ListRecordsByID returns all records by IDs.
func (client *Client) ListRecordsByID(ctx context.Context, zone string, recID string) ([]Record, error) {
	name, tpe, err := parseID(recID)
	if err != nil {
		return nil, err
	}
	return client.ListRecordsInRRSet(ctx, zone, name, tpe)
}

// RecordExists checks if requested record exists in Zone.
func (client *Client) RecordExists(ctx context.Context, zone string, name string, tpe string) (bool, error) {
	allRecords, err := client.ListRecords(ctx, zone)
	if err != nil {
		return false, err
	}

	for _, record := range allRecords {
		if strings.EqualFold(record.Name, name) && strings.EqualFold(record.Type, tpe) {
			return true, nil
		}
	}
	return false, nil
}

// RecordExistsByID checks if requested record exists in Zone by its ID.
func (client *Client) RecordExistsByID(ctx context.Context, zone string, recID string) (bool, error) {
	name, tpe, err := parseID(recID)
	if err != nil {
		return false, err
	}
	return client.RecordExists(ctx, zone, name, tpe)
}

// ReplaceRecordSet creates new record set in Zone.
func (client *Client) ReplaceRecordSet(ctx context.Context, zone string, rrSet ResourceRecordSet) (string, error) {
	rrSet.ChangeType = "REPLACE"

	reqBody, _ := json.Marshal(zonePatchRequest{
		RecordSets: []ResourceRecordSet{rrSet},
	})

	req, err := client.newRequest(ctx, http.MethodPatch, fmt.Sprintf("/servers/localhost/zones/%s", zone), reqBody)
	if err != nil {
		return "", err
	}

	resp, err := client.HTTP.Do(req)
	if err != nil {
		return "", err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			tflog.Warn(ctx, "Error closing response body", map[string]interface{}{
				"error":   err.Error(),
				"method":  req.Method,
				"url":     req.URL.String(),
				"zone":    zone,
				"rrsetId": rrSet.ID(),
			})
		}
	}()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		errorResp := new(errorResponse)
		if err = json.NewDecoder(resp.Body).Decode(errorResp); err != nil {
			return "", fmt.Errorf("error creating record set: %s", rrSet.ID())
		}
		return "", fmt.Errorf("error creating record set: %s, reason: %q", rrSet.ID(), errorResp.ErrorMsg)
	}
	return rrSet.ID(), nil
}

// DeleteRecordSet deletes record set from Zone.
func (client *Client) DeleteRecordSet(ctx context.Context, zone string, name string, tpe string) error {
	reqBody, _ := json.Marshal(zonePatchRequest{
		RecordSets: []ResourceRecordSet{
			{
				Name:       name,
				Type:       tpe,
				ChangeType: "DELETE",
			},
		},
	})

	req, err := client.newRequest(ctx, http.MethodPatch, fmt.Sprintf("/servers/localhost/zones/%s", zone), reqBody)
	if err != nil {
		return err
	}

	resp, err := client.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			tflog.Warn(ctx, "Error closing response body", map[string]interface{}{
				"error":  err.Error(),
				"method": req.Method,
				"url":    req.URL.String(),
				"zone":   zone,
				"name":   name,
				"type":   tpe,
			})
		}
	}()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		errorResp := new(errorResponse)
		if err = json.NewDecoder(resp.Body).Decode(errorResp); err != nil {
			return fmt.Errorf("error deleting record: %s %s", name, tpe)
		}
		return fmt.Errorf("error deleting record: %s %s, reason: %q", name, tpe, errorResp.ErrorMsg)
	}
	return nil
}

// DeleteRecordSetByID deletes record from Zone by its ID.
func (client *Client) DeleteRecordSetByID(ctx context.Context, zone string, recID string) error {
	name, tpe, err := parseID(recID)
	if err != nil {
		return err
	}
	return client.DeleteRecordSet(ctx, zone, name, tpe)
}

func (client *Client) setServerVersion(ctx context.Context) error {
	req, err := client.newRequest(ctx, http.MethodGet, "/servers/localhost", nil)
	if err != nil {
		return err
	}

	resp, err := client.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			tflog.Warn(ctx, "Error closing response body", map[string]interface{}{
				"error":  err.Error(),
				"method": req.Method,
				"url":    req.URL.String(),
			})
		}
	}()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("invalid response code from server: '%d'. Failed to read response body: %v",
				resp.StatusCode, err)
		}
		return fmt.Errorf("failed to set server version: invalid response code from server: '%d'. Response body: %s",
			resp.StatusCode, string(bodyBytes))
	}

	serverInfo := new(serverInfo)
	if err := json.NewDecoder(resp.Body).Decode(serverInfo); err == nil {
		client.ServerVersion = serverInfo.Version
		return nil
	}

	headerServerInfo := strings.SplitN(resp.Header.Get("Server"), "/", 2)
	if len(headerServerInfo) == 2 && strings.EqualFold(headerServerInfo[0], "PowerDNS") {
		client.ServerVersion = headerServerInfo[1]
		return nil
	}

	return fmt.Errorf("unable to get server version")
}

// ListRecursorZones returns all zones of the recursor server.
func (client *Client) ListRecursorZones(ctx context.Context) ([]RecursorZone, error) {
	var zones []RecursorZone
	err := client.doRequestRecursor(ctx, http.MethodGet, "/servers/localhost/zones", nil, http.StatusOK, &zones)
	return zones, err
}

// GetRecursorZone gets a specific zone.
func (client *Client) GetRecursorZone(ctx context.Context, zoneName string) (RecursorZone, error) {
	var zone RecursorZone
	err := client.doRequestRecursor(ctx, http.MethodGet, fmt.Sprintf("/servers/localhost/zones/%s", zoneName), nil, http.StatusOK, &zone)
	return zone, err
}

// CreateRecursorZone creates a new zone.
func (client *Client) CreateRecursorZone(ctx context.Context, zone RecursorZone) (RecursorZone, error) {
	body, err := json.Marshal(zone)
	if err != nil {
		return RecursorZone{}, err
	}

	var createdZone RecursorZone
	err = client.doRequestRecursor(ctx, http.MethodPost, "/servers/localhost/zones", body, http.StatusCreated, &createdZone)
	return createdZone, err
}

// UpdateRecursorZone updates an existing zone using PATCH method.
func (client *Client) UpdateRecursorZone(ctx context.Context, zoneName string, zone RecursorZone) (RecursorZone, error) {
	body, err := json.Marshal(zone)
	if err != nil {
		return RecursorZone{}, err
	}

	var updatedZone RecursorZone
	err = client.doRequestRecursor(ctx, http.MethodPatch, fmt.Sprintf("/servers/localhost/zones/%s", zoneName), body, http.StatusOK, &updatedZone)
	return updatedZone, err
}

// DeleteRecursorZone deletes a zone.
func (client *Client) DeleteRecursorZone(ctx context.Context, zoneName string) error {
	return client.doRequestRecursor(ctx, http.MethodDelete, fmt.Sprintf("/servers/localhost/zones/%s", zoneName), nil, http.StatusNoContent, nil)
}

// doRequest performs a generic HTTP request with common error handling.
func (client *Client) doRequest(ctx context.Context, method, endpoint string, body []byte, successStatus int, response interface{}) error {
	req, err := client.newRequest(ctx, method, endpoint, body)
	if err != nil {
		return err
	}

	resp, err := client.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			tflog.Warn(ctx, "Error closing response body", map[string]interface{}{
				"error":  err.Error(),
				"method": req.Method,
				"url":    req.URL.String(),
			})
		}
	}()

	if resp.StatusCode != successStatus {
		errorResp := new(errorResponse)
		if err = json.NewDecoder(resp.Body).Decode(errorResp); err != nil {
			return fmt.Errorf("error response: %d", resp.StatusCode)
		}
		return fmt.Errorf("error: %d, reason: %q", resp.StatusCode, errorResp.ErrorMsg)
	}

	if response != nil {
		if err := json.NewDecoder(resp.Body).Decode(response); err != nil {
			return err
		}
	}

	return nil
}

// doRequestRecursor performs a generic HTTP request to recursor API with common error handling.
func (client *Client) doRequestRecursor(ctx context.Context, method, endpoint string, body []byte, successStatus int, response interface{}) error {
	req, err := client.newRequestRecursor(ctx, method, endpoint, body)
	if err != nil {
		return err
	}

	resp, err := client.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			tflog.Warn(ctx, "Error closing response body", map[string]interface{}{
				"error":  err.Error(),
				"method": req.Method,
				"url":    req.URL.String(),
			})
		}
	}()

	if resp.StatusCode != successStatus {
		errorResp := new(errorResponse)
		if err = json.NewDecoder(resp.Body).Decode(errorResp); err != nil {
			return fmt.Errorf("error response: %d", resp.StatusCode)
		}
		return fmt.Errorf("error: %d, reason: %q", resp.StatusCode, errorResp.ErrorMsg)
	}

	if response != nil {
		if err := json.NewDecoder(resp.Body).Decode(response); err != nil {
			return err
		}
	}

	return nil
}
