package commands

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/samay58/cairn/internal/source"
)

func runSync(t *testing.T, raw, vault string, extra ...string) syncView {
	t.Helper()
	root := NewRootWithSource(source.NewFixtureSource())
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs(append([]string{"sync", raw, "--to", vault, "--json"}, extra...))
	if err := root.Execute(); err != nil {
		t.Fatalf("sync failed: %v\n%s", err, buf.String())
	}
	var v syncView
	if err := json.Unmarshal(buf.Bytes(), &v); err != nil {
		t.Fatalf("decode sync json: %v\n%s", err, buf.String())
	}
	return v
}

func TestSyncImportsMirrorsAndRelinks(t *testing.T) {
	t.Setenv("CAIRN_HOME", t.TempDir())
	raw := filepath.Join(t.TempDir(), "mymind")
	copySampleExport(t, raw)
	vault := filepath.Join(t.TempDir(), "mymind-cards")

	first := runSync(t, raw, vault, "--from-export")
	if first.Import.Inserted == 0 || first.Export.CardsWritten != first.Import.Inserted {
		t.Fatalf("first sync = %+v; want every inserted card written", first)
	}
	if first.Relink.Linked != 1 || len(first.Relink.Missing) != 0 {
		t.Fatalf("relink = %+v; want the one sample image linked", first.Relink)
	}
	if fi, err := os.Lstat(filepath.Join(raw, "media", "mm_1.png")); err != nil || fi.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("raw image should now be a symlink into the vault")
	}

	// A second run over the linked folder changes nothing.
	second := runSync(t, raw, vault, "--from-export")
	if second.Import.Inserted != 0 || second.Import.Updated != 0 || second.Export.CardsWritten != 0 {
		t.Errorf("second sync = %+v; want no changes", second)
	}
	if second.Relink.Linked != 0 || second.Relink.AlreadyLinked != 1 {
		t.Errorf("second relink = %+v; want 1 already linked", second.Relink)
	}
}

func TestSyncRefusesVaultInsideExportPath(t *testing.T) {
	t.Setenv("CAIRN_HOME", t.TempDir())
	raw := filepath.Join(t.TempDir(), "mymind")
	copySampleExport(t, raw)
	root := NewRootWithSource(source.NewFixtureSource())
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"sync", raw, "--to", raw, "--from-export"})
	if err := root.Execute(); err == nil {
		t.Fatal("sync into the export folder should fail")
	}
}

// fakeMyMind serves two objects and one attachment, rejecting any request
// whose JWT does not verify against secret or names the wrong path.
func fakeMyMind(t *testing.T, secret []byte, blobCalls *int) *httptest.Server {
	t.Helper()
	objects := `[
		{"id": "00000000MDEapiArticle1", "entityType": "Article", "title": "Pacing the frontier",
		 "summary": "[[Anthropic]] argues for slowing down.", "notes": [], "tags": [{"name": "AI"}],
		 "source": {"url": "https://example.com/p?utm_medium=email"}, "created": "2026-09-12T10:00:00.5Z"},
		{"id": "00000000MDEapiDocument", "entityType": "Document", "title": "Safe assets",
		 "blob": {"type": "application/pdf"}, "notes": [], "tags": [], "created": "2026-09-15T10:00:00Z"}
	]`
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !validToken(r, secret) {
			http.Error(w, `{"status":401}`, http.StatusUnauthorized)
			return
		}
		w.Header().Set("RateLimit-Cost", "1")
		w.Header().Set("RateLimit", `"burst";r=9999;t=300, "sustained";r=99990;t=2592000`)
		w.Header().Set("RateLimit-Policy", `"burst";q=10000;w=300, "sustained";q=100000;w=2592000`)
		switch r.URL.Path {
		case "/objects":
			w.Write([]byte(objects))
		case "/objects/00000000MDEapiDocument/blob":
			*blobCalls++
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Write([]byte("%PDF-1.7 safe assets"))
		default:
			http.NotFound(w, r)
		}
	}))
}

func TestSyncPullsFromAPI(t *testing.T) {
	t.Setenv("CAIRN_HOME", t.TempDir())
	secret := []byte("fake-secret-for-tests-0123456789")
	var blobCalls int
	srv := fakeMyMind(t, secret, &blobCalls)
	defer srv.Close()
	t.Setenv("CAIRN_MYMIND_API_URL", srv.URL)
	t.Setenv("CAIRN_MYMIND_KID", "kid-test")
	t.Setenv("CAIRN_MYMIND_SECRET", base64.StdEncoding.EncodeToString(secret))

	raw := filepath.Join(t.TempDir(), "mymind")
	vault := filepath.Join(t.TempDir(), "mymind-cards")
	first := runSync(t, raw, vault)
	if first.API == nil || first.API.Objects != 2 || first.API.AttachmentsFetched != 1 || first.API.SustainedRemaining != 99990 {
		t.Fatalf("api view = %+v", first.API)
	}
	if first.Import.Inserted != 2 || first.Export.CardsWritten != 2 || first.Relink.Linked != 1 {
		t.Fatalf("first sync = %+v", first)
	}
	if _, err := os.Lstat(filepath.Join(raw, "MDEapiDocument.pdf")); err != nil {
		t.Fatalf("attachment not saved under its card id with a .pdf extension: %v", err)
	}
	md, err := os.ReadFile(filepath.Join(vault, "2026-09-12-pacing-the-frontier.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(md), "Anthropic argues for slowing down.") || strings.Contains(string(md), "[[") ||
		!strings.Contains(string(md), "url: https://example.com/p\n") {
		t.Errorf("mirrored card:\n%s", md)
	}

	second := runSync(t, raw, vault)
	if blobCalls != 1 || second.API.AttachmentsFetched != 0 || second.Import.Inserted != 0 || second.Export.CardsWritten != 0 {
		t.Errorf("second sync fetched again or rewrote: blobs %d, %+v", blobCalls, second)
	}
}

func validToken(r *http.Request, secret []byte) bool {
	parts := strings.Split(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "), ".")
	if len(parts) != 3 {
		return false
	}
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(parts[0] + "." + parts[1]))
	claims, _ := base64.RawURLEncoding.DecodeString(parts[1])
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil)) == parts[2] &&
		strings.Contains(string(claims), `"path":"`+r.URL.Path+`"`)
}
