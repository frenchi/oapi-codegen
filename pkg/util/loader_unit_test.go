package util

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

func TestKinLoader_File_Succeeds(t *testing.T) {
	spec := []byte("openapi: 3.0.0\ninfo: {title: t, version: v}\npaths: {}\n")
	dir := t.TempDir()
	p := filepath.Join(dir, "spec.yaml")
	if err := os.WriteFile(p, spec, 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	s, err := kinLoader{}.Load(p)
	if err != nil || s == nil {
		t.Fatalf("kin file load failed: spec=%v err=%v", s != nil, err)
	}
}

func TestKinLoader_URI_Succeeds(t *testing.T) {
	spec := []byte("openapi: 3.0.0\ninfo: {title: t, version: v}\npaths: {}\n")
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/yaml")
		_, _ = w.Write(spec)
	}))
	defer ts.Close()
	s, err := kinLoader{}.Load(ts.URL + "/spec.yaml")
	if err != nil || s == nil {
		t.Fatalf("kin uri load failed: spec=%v err=%v", s != nil, err)
	}
}

func TestKinLoader_URI_Invalid_Fails(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/yaml")
		_, _ = w.Write([]byte("not: [valid"))
	}))
	defer ts.Close()
	if _, err := (kinLoader{}.Load(ts.URL + "/bad.yaml")); err == nil {
		t.Fatalf("expected error for invalid uri spec")
	}
}

func TestKinLoader_File_Invalid_Fails(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(p, []byte("not: [valid"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := (kinLoader{}.Load(p)); err == nil {
		t.Fatalf("expected error for invalid file")
	}
}

func TestLibopenapiStrict_File_Succeeds(t *testing.T) {
	spec := []byte("openapi: 3.0.0\ninfo: {title: t, version: v}\npaths: {}\n")
	dir := t.TempDir()
	p := filepath.Join(dir, "spec.yaml")
	if err := os.WriteFile(p, spec, 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	s, err := libopenapiLoaderStrict{}.Load(p)
	if err != nil || s == nil {
		t.Fatalf("lib strict file load failed: spec=%v err=%v", s != nil, err)
	}
}

func TestLibopenapiStrict_URI_Succeeds(t *testing.T) {
	spec := []byte("openapi: 3.0.0\ninfo: {title: t, version: v}\npaths: {}\n")
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/yaml")
		_, _ = w.Write(spec)
	}))
	defer ts.Close()
	s, err := libopenapiLoaderStrict{}.Load(ts.URL + "/spec.yaml")
	if err != nil || s == nil {
		t.Fatalf("lib strict uri load failed: spec=%v err=%v", s != nil, err)
	}
}

func TestLibopenapiStrict_File_Invalid_Fails(t *testing.T) {
	dir := t.TempDir()
	bad := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(bad, []byte("not: [valid"), 0o600); err != nil {
		t.Fatalf("write bad: %v", err)
	}
	if _, err := (libopenapiLoaderStrict{}.Load(bad)); err == nil {
		t.Fatalf("expected error for invalid file")
	}
}

func TestLibopenapiStrict_URI_Invalid_Fails(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/yaml")
		_, _ = w.Write([]byte("not: [valid"))
	}))
	defer ts.Close()
	_, err := libopenapiLoaderStrict{}.Load(ts.URL + "/bad.yaml")
	if err == nil {
		t.Fatalf("expected error for invalid uri spec")
	}
}

func Test_libopenapiLoadFromURI_Succeeds(t *testing.T) {
	good := []byte("openapi: 3.0.0\ninfo: {title: t, version: v}\npaths: {}\n")

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/yaml")
		_, _ = w.Write(good)
	}))
	defer ts.Close()

	u, _ := url.Parse(ts.URL + "/good.yaml")
	if s, err := libopenapiLoadFromURI(u); err != nil || s == nil {
		t.Fatalf("expected success for good uri, err=%v", err)
	}
}

func Test_libopenapiLoadFromURI_Fails(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/yaml")
		_, _ = w.Write([]byte("not: [valid"))
	}))
	defer ts.Close()

	u, _ := url.Parse(ts.URL + "/bad.yaml")
	if _, err := libopenapiLoadFromURI(u); err == nil {
		t.Fatalf("expected error for bad uri")
	}
}

func Test_libopenapiLoadFromFile_Succeeds(t *testing.T) {
	dir := t.TempDir()
	good := filepath.Join(dir, "good.yaml")
	if err := os.WriteFile(good, []byte("openapi: 3.0.0\ninfo: {title: t, version: v}\npaths: {}\n"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	if s, err := libopenapiLoadFromFile(good); err != nil || s == nil {
		t.Fatalf("expected success for good file, err=%v", err)
	}
}

func Test_libopenapiLoadFromFile_Fails(t *testing.T) {
	dir := t.TempDir()
	bad := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(bad, []byte("not: [valid"), 0o600); err != nil {
		t.Fatalf("write bad: %v", err)
	}
	if _, err := libopenapiLoadFromFile(bad); err == nil {
		t.Fatalf("expected error for bad file")
	}
}
