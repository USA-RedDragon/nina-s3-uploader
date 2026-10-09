package uploader_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/USA-RedDragon/nina-s3-uploader/internal/config"
	"github.com/USA-RedDragon/nina-s3-uploader/internal/uploader"
)

type fakeS3 struct {
	mu      sync.Mutex
	objects map[string]string
}

func (f *fakeS3) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	key := strings.TrimPrefix(r.URL.Path, "/bucket/")
	switch r.Method {
	case http.MethodPut:
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		f.objects[key] = string(body)
	case http.MethodHead:
		if _, ok := f.objects[key]; !ok {
			w.WriteHeader(http.StatusNotFound)
		}
	default:
		w.WriteHeader(http.StatusNotImplemented)
	}
}

func (f *fakeS3) keys() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	keys := make([]string, 0, len(f.objects))
	for k := range f.objects {
		keys = append(keys, k)
	}
	return keys
}

func newUploader(t *testing.T, dir, local string) (*uploader.Uploader, *fakeS3) {
	t.Helper()
	fake := &fakeS3{objects: map[string]string{}}
	srv := httptest.NewServer(fake)
	t.Cleanup(srv.Close)
	cfg := &config.Config{
		S3: config.S3{Region: "us-east-1", Bucket: "bucket", Prefix: "astro", Endpoint: srv.URL},
		Uploader: config.Uploader{
			Directory:  dir,
			Extensions: []string{".fits"},
			Local:      config.Local{Directory: local},
		},
	}
	u, err := uploader.NewUploader(cfg)
	if err != nil {
		t.Fatal(err)
	}
	return u, fake
}

func writeFile(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("data"), 0o600); err != nil {
		t.Fatal(err)
	}
}

func expectKeys(t *testing.T, fake *fakeS3, want ...string) {
	t.Helper()
	got := fake.keys()
	if len(got) != len(want) {
		t.Fatalf("uploaded %q, want %q", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("uploaded %q, want %q", got, want)
		}
	}
}

func TestUploadWithDotSlashDirectory(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "id")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "secret")
	t.Setenv("AWS_CONFIG_FILE", "/nonexistent")
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", "/nonexistent")
	t.Chdir(t.TempDir())
	u, fake := newUploader(t, "./src", "./local")
	writeFile(t, filepath.Join("src", "night", "light.fits"))
	if err := u.Upload(filepath.Join("src", "night", "light.fits")); err != nil {
		t.Fatal(err)
	}
	expectKeys(t, fake, "astro/night/light.fits")
}

func TestUploadAbsolutePathWithRelativeLocalDirectory(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "id")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "secret")
	t.Setenv("AWS_CONFIG_FILE", "/nonexistent")
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", "/nonexistent")
	root := t.TempDir()
	t.Chdir(root)
	u, fake := newUploader(t, "src", "local")
	path := filepath.Join(root, "local", "night", "light.fits")
	writeFile(t, path)
	if err := u.Upload(path); err != nil {
		t.Fatal(err)
	}
	expectKeys(t, fake, "astro/night/light.fits")
}

func TestUploadDoesNotMatchSiblingDirectoryPrefix(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "id")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "secret")
	t.Setenv("AWS_CONFIG_FILE", "/nonexistent")
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", "/nonexistent")
	root := t.TempDir()
	u, fake := newUploader(t, filepath.Join(root, "data2"), filepath.Join(root, "data"))
	path := filepath.Join(root, "data2", "light.fits")
	writeFile(t, path)
	if err := u.Upload(path); err != nil {
		t.Fatal(err)
	}
	expectKeys(t, fake, "astro/light.fits")
}

func TestUploadOutsideDirectoriesFails(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "id")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "secret")
	t.Setenv("AWS_CONFIG_FILE", "/nonexistent")
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", "/nonexistent")
	root := t.TempDir()
	u, fake := newUploader(t, filepath.Join(root, "src"), filepath.Join(root, "local"))
	path := filepath.Join(root, "elsewhere", "light.fits")
	writeFile(t, path)
	if err := u.Upload(path); err == nil {
		t.Fatal("expected an error for a file outside the configured directories")
	}
	expectKeys(t, fake)
}
