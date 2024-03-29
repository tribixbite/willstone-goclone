// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package http_test

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"mime/multipart"
	"net"
	. "net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"
)

const (
	testFile    = "testdata/file"
	testFileLen = 11
)

type wantRange struct {
	start, end int64 // range [start,end)
}

var ServeFileRangeTests = []struct {
	r      string
	code   int
	ranges []wantRange
}{
	{r: "", code: StatusOK},
	{r: "bytes=0-4", code: StatusPartialContent, ranges: []wantRange{{0, 5}}},
	{r: "bytes=2-", code: StatusPartialContent, ranges: []wantRange{{2, testFileLen}}},
	{r: "bytes=-5", code: StatusPartialContent, ranges: []wantRange{{testFileLen - 5, testFileLen}}},
	{r: "bytes=3-7", code: StatusPartialContent, ranges: []wantRange{{3, 8}}},
	{r: "bytes=20-", code: StatusRequestedRangeNotSatisfiable},
	{r: "bytes=0-0,-2", code: StatusPartialContent, ranges: []wantRange{{0, 1}, {testFileLen - 2, testFileLen}}},
	{r: "bytes=0-1,5-8", code: StatusPartialContent, ranges: []wantRange{{0, 2}, {5, 9}}},
	{r: "bytes=0-1,5-", code: StatusPartialContent, ranges: []wantRange{{0, 2}, {5, testFileLen}}},
	{r: "bytes=0-,1-,2-,3-,4-", code: StatusOK}, // ignore wasteful range request
}

func TestServeFile(t *testing.T) {
	ts := httptest.NewServer(HandlerFunc(func(w ResponseWriter, r *Request) {
		ServeFile(w, r, "testdata/file")
	}))
	defer ts.Close()

	var err error

	file, err := ioutil.ReadFile(testFile)
	if err != nil {
		t.Fatal("reading file:", err)
	}

	// set up the Request (re-used for all tests)
	var req Request
	req.Header = make(Header)
	if req.URL, err = url.Parse(ts.URL); err != nil {
		t.Fatal("ParseURL:", err)
	}
	req.Method = "GET"

	// straight GET
	_, body := getBody(t, "straight get", req)
	if !bytes.Equal(body, file) {
		t.Fatalf("body mismatch: got %q, want %q", body, file)
	}

	// Range tests
Cases:
	for _, rt := range ServeFileRangeTests {
		if rt.r != "" {
			req.Header.Set("Range", rt.r)
		}
		resp, body := getBody(t, fmt.Sprintf("range test %q", rt.r), req)
		if resp.StatusCode != rt.code {
			t.Errorf("range=%q: StatusCode=%d, want %d", rt.r, resp.StatusCode, rt.code)
		}
		if rt.code == StatusRequestedRangeNotSatisfiable {
			continue
		}
		wantContentRange := ""
		if len(rt.ranges) == 1 {
			rng := rt.ranges[0]
			wantContentRange = fmt.Sprintf("bytes %d-%d/%d", rng.start, rng.end-1, testFileLen)
		}
		cr := resp.Header.Get("Content-Range")
		if cr != wantContentRange {
			t.Errorf("range=%q: Content-Range = %q, want %q", rt.r, cr, wantContentRange)
		}
		ct := resp.Header.Get("Content-Type")
		if len(rt.ranges) == 1 {
			rng := rt.ranges[0]
			wantBody := file[rng.start:rng.end]
			if !bytes.Equal(body, wantBody) {
				t.Errorf("range=%q: body = %q, want %q", rt.r, body, wantBody)
			}
			if strings.HasPrefix(ct, "multipart/byteranges") {
				t.Errorf("range=%q content-type = %q; unexpected multipart/byteranges", rt.r, ct)
			}
		}
		if len(rt.ranges) > 1 {
			typ, params, err := mime.ParseMediaType(ct)
			if err != nil {
				t.Errorf("range=%q content-type = %q; %v", rt.r, ct, err)
				continue
			}
			if typ != "multipart/byteranges" {
				t.Errorf("range=%q content-type = %q; want multipart/byteranges", rt.r, typ)
				continue
			}
			if params["boundary"] == "" {
				t.Errorf("range=%q content-type = %q; lacks boundary", rt.r, ct)
				continue
			}
			if g, w := resp.ContentLength, int64(len(body)); g != w {
				t.Errorf("range=%q Content-Length = %d; want %d", rt.r, g, w)
				continue
			}
			mr := multipart.NewReader(bytes.NewReader(body), params["boundary"])
			for ri, rng := range rt.ranges {
				part, err := mr.NextPart()
				if err != nil {
					t.Errorf("range=%q, reading part index %d: %v", rt.r, ri, err)
					continue Cases
				}
				wantContentRange = fmt.Sprintf("bytes %d-%d/%d", rng.start, rng.end-1, testFileLen)
				if g, w := part.Header.Get("Content-Range"), wantContentRange; g != w {
					t.Errorf("range=%q: part Content-Range = %q; want %q", rt.r, g, w)
				}
				body, err := ioutil.ReadAll(part)
				if err != nil {
					t.Errorf("range=%q, reading part index %d body: %v", rt.r, ri, err)
					continue Cases
				}
				wantBody := file[rng.start:rng.end]
				if !bytes.Equal(body, wantBody) {
					t.Errorf("range=%q: body = %q, want %q", rt.r, body, wantBody)
				}
			}
			_, err = mr.NextPart()
			if err != io.EOF {
				t.Errorf("range=%q; expected final error io.EOF; got %v", rt.r, err)
			}
		}
	}
}

var fsRedirectTestData = []struct {
	original, redirect string
}{
	{"/test/index.html", "/test/"},
	{"/test/testdata", "/test/testdata/"},
	{"/test/testdata/file/", "/test/testdata/file"},
}

func TestFSRedirect(t *testing.T) {
	ts := httptest.NewServer(StripPrefix("/test", FileServer(Dir("."))))
	defer ts.Close()

	for _, data := range fsRedirectTestData {
		res, err := Get(ts.URL + data.original)
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		if g, e := res.Request.URL.Path, data.redirect; g != e {
			t.Errorf("redirect from %s: got %s, want %s", data.original, g, e)
		}
	}
}

type testFileSystem struct {
	open func(name string) (File, error)
}

func (fs *testFileSystem) Open(name string) (File, error) {
	return fs.open(name)
}

func TestFileServerCleans(t *testing.T) {
	ch := make(chan string, 1)
	fs := FileServer(&testFileSystem{func(name string) (File, error) {
		ch <- name
		return nil, errors.New("file does not exist")
	}})
	tests := []struct {
		reqPath, openArg string
	}{
		{"/foo.txt", "/foo.txt"},
		{"//foo.txt", "/foo.txt"},
		{"/../foo.txt", "/foo.txt"},
	}
	req, _ := NewRequest("GET", "http://example.com", nil)
	for n, test := range tests {
		rec := httptest.NewRecorder()
		req.URL.Path = test.reqPath
		fs.ServeHTTP(rec, req)
		if got := <-ch; got != test.openArg {
			t.Errorf("test %d: got %q, want %q", n, got, test.openArg)
		}
	}
}

func mustRemoveAll(dir string) {
	err := os.RemoveAll(dir)
	if err != nil {
		panic(err)
	}
}

func TestFileServerImplicitLeadingSlash(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("TempDir: %v", err)
	}
	defer mustRemoveAll(tempDir)
	if err := ioutil.WriteFile(filepath.Join(tempDir, "foo.txt"), []byte("Hello world"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	ts := httptest.NewServer(StripPrefix("/bar/", FileServer(Dir(tempDir))))
	defer ts.Close()
	get := func(suffix string) string {
		res, err := Get(ts.URL + suffix)
		if err != nil {
			t.Fatalf("Get %s: %v", suffix, err)
		}
		b, err := ioutil.ReadAll(res.Body)
		if err != nil {
			t.Fatalf("ReadAll %s: %v", suffix, err)
		}
		res.Body.Close()
		return string(b)
	}
	if s := get("/bar/"); !strings.Contains(s, ">foo.txt<") {
		t.Logf("expected a directory listing with foo.txt, got %q", s)
	}
	if s := get("/bar/foo.txt"); s != "Hello world" {
		t.Logf("expected %q, got %q", "Hello world", s)
	}
}

func TestDirJoin(t *testing.T) {
	wfi, err := os.Stat("/etc/hosts")
	if err != nil {
		t.Logf("skipping test; no /etc/hosts file")
		return
	}
	test := func(d Dir, name string) {
		f, err := d.Open(name)
		if err != nil {
			t.Fatalf("open of %s: %v", name, err)
		}
		defer f.Close()
		gfi, err := f.Stat()
		if err != nil {
			t.Fatalf("stat of %s: %v", name, err)
		}
		if !os.SameFile(gfi, wfi) {
			t.Errorf("%s got different file", name)
		}
	}
	test(Dir("/etc/"), "/hosts")
	test(Dir("/etc/"), "hosts")
	test(Dir("/etc/"), "../../../../hosts")
	test(Dir("/etc"), "/hosts")
	test(Dir("/etc"), "hosts")
	test(Dir("/etc"), "../../../../hosts")

	// Not really directories, but since we use this trick in
	// ServeFile, test it:
	test(Dir("/etc/hosts"), "")
	test(Dir("/etc/hosts"), "/")
	test(Dir("/etc/hosts"), "../")
}

func TestEmptyDirOpenCWD(t *testing.T) {
	test := func(d Dir) {
		name := "fs_test.go"
		f, err := d.Open(name)
		if err != nil {
			t.Fatalf("open of %s: %v", name, err)
		}
		defer f.Close()
	}
	test(Dir(""))
	test(Dir("."))
	test(Dir("./"))
}

func TestServeFileContentType(t *testing.T) {
	const ctype = "icecream/chocolate"
	ts := httptest.NewServer(HandlerFunc(func(w ResponseWriter, r *Request) {
		if r.FormValue("override") == "1" {
			w.Header().Set("Content-Type", ctype)
		}
		ServeFile(w, r, "testdata/file")
	}))
	defer ts.Close()
	get := func(override, want string) {
		resp, err := Get(ts.URL + "?override=" + override)
		if err != nil {
			t.Fatal(err)
		}
		if h := resp.Header.Get("Content-Type"); h != want {
			t.Errorf("Content-Type mismatch: got %q, want %q", h, want)
		}
	}
	get("0", "text/plain; charset=utf-8")
	get("1", ctype)
}

func TestServeFileMimeType(t *testing.T) {
	ts := httptest.NewServer(HandlerFunc(func(w ResponseWriter, r *Request) {
		ServeFile(w, r, "testdata/style.css")
	}))
	defer ts.Close()
	resp, err := Get(ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	want := "text/css; charset=utf-8"
	if h := resp.Header.Get("Content-Type"); h != want {
		t.Errorf("Content-Type mismatch: got %q, want %q", h, want)
	}
}

func TestServeFileFromCWD(t *testing.T) {
	ts := httptest.NewServer(HandlerFunc(func(w ResponseWriter, r *Request) {
		ServeFile(w, r, "fs_test.go")
	}))
	defer ts.Close()
	r, err := Get(ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	if r.StatusCode != 200 {
		t.Fatalf("expected 200 OK, got %s", r.Status)
	}
}

func TestServeFileWithContentEncoding(t *testing.T) {
	ts := httptest.NewServer(HandlerFunc(func(w ResponseWriter, r *Request) {
		w.Header().Set("Content-Encoding", "foo")
		ServeFile(w, r, "testdata/file")
	}))
	defer ts.Close()
	resp, err := Get(ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	if g, e := resp.ContentLength, int64(-1); g != e {
		t.Errorf("Content-Length mismatch: got %d, want %d", g, e)
	}
}

func TestServeIndexHtml(t *testing.T) {
	const want = "index.html says hello\n"
	ts := httptest.NewServer(FileServer(Dir(".")))
	defer ts.Close()

	for _, path := range []string{"/testdata/", "/testdata/index.html"} {
		res, err := Get(ts.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		b, err := ioutil.ReadAll(res.Body)
		if err != nil {
			t.Fatal("reading Body:", err)
		}
		if s := string(b); s != want {
			t.Errorf("for path %q got %q, want %q", path, s, want)
		}
		res.Body.Close()
	}
}

type fakeFileInfo struct {
	dir      bool
	basename string
	modtime  time.Time
	ents     []*fakeFileInfo
	contents string
}

func (f *fakeFileInfo) Name() string       { return f.basename }
func (f *fakeFileInfo) Sys() interface{}   { return nil }
func (f *fakeFileInfo) ModTime() time.Time { return f.modtime }
func (f *fakeFileInfo) IsDir() bool        { return f.dir }
func (f *fakeFileInfo) Size() int64        { return int64(len(f.contents)) }
func (f *fakeFileInfo) Mode() os.FileMode {
	if f.dir {
		return 0755 | os.ModeDir
	}
	return 0644
}

type fakeFile struct {
	io.ReadSeeker
	fi   *fakeFileInfo
	path string // as opened
}

func (f *fakeFile) Close() error               { return nil }
func (f *fakeFile) Stat() (os.FileInfo, error) { return f.fi, nil }
func (f *fakeFile) Readdir(count int) ([]os.FileInfo, error) {
	if !f.fi.dir {
		return nil, os.ErrInvalid
	}
	var fis []os.FileInfo
	for _, fi := range f.fi.ents {
		fis = append(fis, fi)
	}
	return fis, nil
}

type fakeFS map[string]*fakeFileInfo

func (fs fakeFS) Open(name string) (File, error) {
	name = path.Clean(name)
	f, ok := fs[name]
	if !ok {
		println("fake filesystem didn't find file", name)
		return nil, os.ErrNotExist
	}
	return &fakeFile{ReadSeeker: strings.NewReader(f.contents), fi: f, path: name}, nil
}

func TestDirectoryIfNotModified(t *testing.T) {
	const indexContents = "I am a fake index.html file"
	fileMod := time.Unix(1000000000, 0).UTC()
	fileModStr := fileMod.Format(TimeFormat)
	dirMod := time.Unix(123, 0).UTC()
	indexFile := &fakeFileInfo{
		basename: "index.html",
		modtime:  fileMod,
		contents: indexContents,
	}
	fs := fakeFS{
		"/": &fakeFileInfo{
			dir:     true,
			modtime: dirMod,
			ents:    []*fakeFileInfo{indexFile},
		},
		"/index.html": indexFile,
	}

	ts := httptest.NewServer(FileServer(fs))
	defer ts.Close()

	res, err := Get(ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != indexContents {
		t.Fatalf("Got body %q; want %q", b, indexContents)
	}
	res.Body.Close()

	lastMod := res.Header.Get("Last-Modified")
	if lastMod != fileModStr {
		t.Fatalf("initial Last-Modified = %q; want %q", lastMod, fileModStr)
	}

	req, _ := NewRequest("GET", ts.URL, nil)
	req.Header.Set("If-Modified-Since", lastMod)

	res, err = DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != 304 {
		t.Fatalf("Code after If-Modified-Since request = %v; want 304", res.StatusCode)
	}
	res.Body.Close()

	// Advance the index.html file's modtime, but not the directory's.
	indexFile.modtime = indexFile.modtime.Add(1 * time.Hour)

	res, err = DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != 200 {
		t.Fatalf("Code after second If-Modified-Since request = %v; want 200; res is %#v", res.StatusCode, res)
	}
	res.Body.Close()
}

func TestServeContent(t *testing.T) {
	type req struct {
		name    string
		modtime time.Time
		content io.ReadSeeker
	}
	ch := make(chan req, 1)
	ts := httptest.NewServer(HandlerFunc(func(w ResponseWriter, r *Request) {
		p := <-ch
		ServeContent(w, r, p.name, p.modtime, p.content)
	}))
	defer ts.Close()

	css, err := os.Open("testdata/style.css")
	if err != nil {
		t.Fatal(err)
	}
	defer css.Close()

	ch <- req{"style.css", time.Time{}, css}
	res, err := Get(ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	if g, e := res.Header.Get("Content-Type"), "text/css; charset=utf-8"; g != e {
		t.Errorf("style.css: content type = %q, want %q", g, e)
	}
	if g := res.Header.Get("Last-Modified"); g != "" {
		t.Errorf("want empty Last-Modified; got %q", g)
	}

	fi, err := css.Stat()
	if err != nil {
		t.Fatal(err)
	}
	ch <- req{"style.html", fi.ModTime(), css}
	res, err = Get(ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	if g, e := res.Header.Get("Content-Type"), "text/html; charset=utf-8"; g != e {
		t.Errorf("style.html: content type = %q, want %q", g, e)
	}
	if g := res.Header.Get("Last-Modified"); g == "" {
		t.Errorf("want non-empty last-modified")
	}
}

// verifies that sendfile is being used on Linux
func TestLinuxSendfile(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Logf("skipping; linux-only test")
		return
	}
	_, err := exec.LookPath("strace")
	if err != nil {
		t.Logf("skipping; strace not found in path")
		return
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	lnf, err := ln.(*net.TCPListener).File()
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	var buf bytes.Buffer
	child := exec.Command("strace", "-f", os.Args[0], "-test.run=TestLinuxSendfileChild")
	child.ExtraFiles = append(child.ExtraFiles, lnf)
	child.Env = append([]string{"GO_WANT_HELPER_PROCESS=1"}, os.Environ()...)
	child.Stdout = &buf
	child.Stderr = &buf
	err = child.Start()
	if err != nil {
		t.Logf("skipping; failed to start straced child: %v", err)
		return
	}

	res, err := Get(fmt.Sprintf("http://%s/", ln.Addr()))
	if err != nil {
		t.Fatalf("http client error: %v", err)
	}
	_, err = io.Copy(ioutil.Discard, res.Body)
	if err != nil {
		t.Fatalf("client body read error: %v", err)
	}
	res.Body.Close()

	// Force child to exit cleanly.
	Get(fmt.Sprintf("http://%s/quit", ln.Addr()))
	child.Wait()

	rx := regexp.MustCompile(`sendfile(64)?\(\d+,\s*\d+,\s*NULL,\s*\d+\)\s*=\s*\d+\s*\n`)
	rxResume := regexp.MustCompile(`<\.\.\. sendfile(64)? resumed> \)\s*=\s*\d+\s*\n`)
	out := buf.String()
	if !rx.MatchString(out) && !rxResume.MatchString(out) {
		t.Errorf("no sendfile system call found in:\n%s", out)
	}
}

func getBody(t *testing.T, testName string, req Request) (*Response, []byte) {
	r, err := DefaultClient.Do(&req)
	if err != nil {
		t.Fatalf("%s: for URL %q, send error: %v", testName, req.URL.String(), err)
	}
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		t.Fatalf("%s: for URL %q, reading body: %v", testName, req.URL.String(), err)
	}
	return r, b
}

// TestLinuxSendfileChild isn't a real test. It's used as a helper process
// for TestLinuxSendfile.
func TestLinuxSendfileChild(*testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	defer os.Exit(0)
	fd3 := os.NewFile(3, "ephemeral-port-listener")
	ln, err := net.FileListener(fd3)
	if err != nil {
		panic(err)
	}
	mux := NewServeMux()
	mux.Handle("/", FileServer(Dir("testdata")))
	mux.HandleFunc("/quit", func(ResponseWriter, *Request) {
		os.Exit(0)
	})
	s := &Server{Handler: mux}
	err = s.Serve(ln)
	if err != nil {
		panic(err)
	}
}
