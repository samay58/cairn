// Package mymindapi reads a MyMind library through the official API
// (https://access.mymind.com/api, public beta since June 2026). Cairn only
// reads, so a READ ONLY key is enough.
package mymindapi

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const DefaultBaseURL = "https://api.mymind.com"

// Keychain service names checked when the environment carries no key.
const (
	KeychainKeyID  = "mymind-api-kid"
	KeychainSecret = "mymind-api-secret"
)

// ErrNoCredentials means neither the environment nor the Keychain holds a key.
var ErrNoCredentials = errors.New("no MyMind API key: set CAIRN_MYMIND_KID and CAIRN_MYMIND_SECRET, or add Keychain items " + KeychainKeyID + " and " + KeychainSecret)

type Credentials struct {
	KeyID  string
	Secret []byte // decoded from the base64 secret MyMind issues
}

// LoadCredentials reads CAIRN_MYMIND_KID and CAIRN_MYMIND_SECRET, falling back
// to the macOS Keychain.
func LoadCredentials() (Credentials, error) {
	kid, secret := os.Getenv("CAIRN_MYMIND_KID"), os.Getenv("CAIRN_MYMIND_SECRET")
	if kid == "" || secret == "" {
		kid, secret = keychain(KeychainKeyID), keychain(KeychainSecret)
	}
	if kid == "" || secret == "" {
		return Credentials{}, ErrNoCredentials
	}
	raw, err := base64.StdEncoding.DecodeString(secret)
	if err != nil {
		return Credentials{}, fmt.Errorf("MyMind API secret is not base64: %w", err)
	}
	return Credentials{KeyID: kid, Secret: raw}, nil
}

func keychain(service string) string {
	out, err := exec.Command("security", "find-generic-password", "-s", service, "-w").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// Quota is the credit state MyMind reports after a request.
type Quota struct {
	Cost               int `json:"cost"`
	SustainedRemaining int `json:"sustained_remaining"`
	SustainedLimit     int `json:"sustained_limit"`
	BurstResetSeconds  int `json:"-"`
	SustainedResetSecs int `json:"-"`
}

type Client struct {
	BaseURL string
	Creds   Credentials
	HTTP    *http.Client
	Now     func() time.Time
}

// NewClient returns a client for the MyMind API, or for CAIRN_MYMIND_API_URL
// when set (tests point it at a local server).
func NewClient(creds Credentials) *Client {
	base := DefaultBaseURL
	if u := os.Getenv("CAIRN_MYMIND_API_URL"); u != "" {
		base = strings.TrimRight(u, "/")
	}
	return &Client{BaseURL: base, Creds: creds, HTTP: &http.Client{Timeout: 2 * time.Minute}, Now: time.Now}
}

// ListObjects returns every object in the library. One request, about one
// credit, whatever the library size up to MyMind's 10,000-object page.
func (c *Client) ListObjects(ctx context.Context) ([]Object, Quota, error) {
	body, _, q, err := c.get(ctx, "/objects", "limit=10000")
	if err != nil {
		return nil, q, err
	}
	var objs []Object
	if err := json.Unmarshal(body, &objs); err != nil {
		return nil, q, fmt.Errorf("decode objects: %w", err)
	}
	return objs, q, nil
}

// Blob downloads an object's stored file and its media type. One credit.
func (c *Client) Blob(ctx context.Context, id string) ([]byte, string, Quota, error) {
	return c.get(ctx, "/objects/"+id+"/blob", "")
}

func (c *Client) get(ctx context.Context, path, query string) ([]byte, string, Quota, error) {
	url := c.BaseURL + path
	if query != "" {
		url += "?" + query
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, "", Quota{}, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token(http.MethodGet, path))
	req.Header.Set("User-Agent", "cairn")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, "", Quota{}, fmt.Errorf("GET %s: %w", path, err)
	}
	defer resp.Body.Close()
	q := parseQuota(resp.Header)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", q, fmt.Errorf("GET %s: %w", path, err)
	}
	switch {
	case resp.StatusCode == http.StatusTooManyRequests:
		return nil, "", q, fmt.Errorf("MyMind API quota exhausted; it resets in %s", time.Duration(max(q.BurstResetSeconds, q.SustainedResetSecs))*time.Second)
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		return nil, "", q, fmt.Errorf("MyMind API rejected the key (HTTP %d): %s", resp.StatusCode, brief(body))
	case resp.StatusCode != http.StatusOK:
		return nil, "", q, fmt.Errorf("GET %s: HTTP %d: %s", path, resp.StatusCode, brief(body))
	}
	return body, resp.Header.Get("Content-Type"), q, nil
}

// token signs one request: an HS256 JWT whose header names the key and whose
// claims bind the method and path, valid for five minutes.
func (c *Client) token(method, path string) string {
	enc := base64.RawURLEncoding
	now := c.Now().Unix()
	head, _ := json.Marshal(map[string]string{"alg": "HS256", "typ": "JWT", "kid": c.Creds.KeyID})
	claims, _ := json.Marshal(map[string]any{"method": method, "path": path, "iat": now, "exp": now + 300})
	unsigned := enc.EncodeToString(head) + "." + enc.EncodeToString(claims)
	mac := hmac.New(sha256.New, c.Creds.Secret)
	mac.Write([]byte(unsigned))
	return unsigned + "." + enc.EncodeToString(mac.Sum(nil))
}

var policyParam = regexp.MustCompile(`"(\w+)"((?:;\w+=\d+)+)`)

// parseQuota reads RateLimit, RateLimit-Policy and RateLimit-Cost, e.g.
// RateLimit: "burst";r=9967;t=261, "sustained";r=99967;t=2591961
func parseQuota(h http.Header) Quota {
	var q Quota
	q.Cost, _ = strconv.Atoi(h.Get("RateLimit-Cost"))
	for _, m := range policyParam.FindAllStringSubmatch(h.Get("RateLimit"), -1) {
		r, t := param(m[2], "r"), param(m[2], "t")
		if m[1] == "sustained" {
			q.SustainedRemaining, q.SustainedResetSecs = r, t
		} else if m[1] == "burst" {
			q.BurstResetSeconds = t
		}
	}
	for _, m := range policyParam.FindAllStringSubmatch(h.Get("RateLimit-Policy"), -1) {
		if m[1] == "sustained" {
			q.SustainedLimit = param(m[2], "q")
		}
	}
	return q
}

func param(params, key string) int {
	for _, kv := range strings.Split(strings.TrimPrefix(params, ";"), ";") {
		if k, v, ok := strings.Cut(kv, "="); ok && k == key {
			n, _ := strconv.Atoi(v)
			return n
		}
	}
	return 0
}

// brief flattens an error body (MyMind returns pretty-printed JSON problems)
// to one short line.
func brief(b []byte) string {
	s := strings.Join(strings.Fields(string(b)), " ")
	if len(s) > 200 {
		s = s[:200]
	}
	return s
}
