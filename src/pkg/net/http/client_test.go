// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Tests for client.go

package http_test

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	. "net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"
)

var robotsTxtHandler = HandlerFunc(func(w ResponseWriter, r *Request) {
	w.Header().Set("Last-Modified", "sometime")
	fmt.Fprintf(w, "User-agent: go\nDisallow: /something/")
})

// pedanticReadAll works like ioutil.ReadAll but additionally
// verifies that r obeys the documented io.Reader contract.
func pedanticReadAll(r io.Reader) (b []byte, err error) {
	var bufa [64]byte
	buf := bufa[:]
	for {
		n, err := r.Read(buf)
		if n == 0 && err == nil {
			return nil, fmt.Errorf("Read: n=0 with err=nil")
		}
		b = append(b, buf[:n]...)
		if err == io.EOF {
			n, err := r.Read(buf)
			if n != 0 || err != io.EOF {
				return nil, fmt.Errorf("Read: n=%d err=%#v after EOF", n, err)
			}
			return b, nil
		}
		if err != nil {
			return b, err
		}
	}
	panic("unreachable")
}

func TestClient(t *testing.T) {
	ts := httptest.NewServer(robotsTxtHandler)
	defer ts.Close()

	r, err := Get(ts.URL)
	var b []byte
	if err == nil {
		b, err = pedanticReadAll(r.Body)
		r.Body.Close()
	}
	if err != nil {
		t.Error(err)
	} else if s := string(b); !strings.HasPrefix(s, "User-agent:") {
		t.Errorf("Incorrect page body (did not begin with User-agent): %q", s)
	}
}

func TestClientHead(t *testing.T) {
	ts := httptest.NewServer(robotsTxtHandler)
	defer ts.Close()

	r, err := Head(ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := r.Header["Last-Modified"]; !ok {
		t.Error("Last-Modified header not found.")
	}
}

type recordingTransport struct {
	req *Request
}

func (t *recordingTransport) RoundTrip(req *Request) (resp *Response, err error) {
	t.req = req
	return nil, errors.New("dummy impl")
}

func TestGetRequestFormat(t *testing.T) {
	tr := &recordingTransport{}
	client := &Client{Transport: tr}
	url := "http://dummy.faketld/"
	client.Get(url) // Note: doesn't hit network
	if tr.req.Method != "GET" {
		t.Errorf("expected method %q; got %q", "GET", tr.req.Method)
	}
	if tr.req.URL.String() != url {
		t.Errorf("expected URL %q; got %q", url, tr.req.URL.String())
	}
	if tr.req.Header == nil {
		t.Errorf("expected non-nil request Header")
	}
}

func TestPostRequestFormat(t *testing.T) {
	tr := &recordingTransport{}
	client := &Client{Transport: tr}

	url := "http://dummy.faketld/"
	json := `{"key":"value"}`
	b := strings.NewReader(json)
	client.Post(url, "application/json", b) // Note: doesn't hit network

	if tr.req.Method != "POST" {
		t.Errorf("got method %q, want %q", tr.req.Method, "POST")
	}
	if tr.req.URL.String() != url {
		t.Errorf("got URL %q, want %q", tr.req.URL.String(), url)
	}
	if tr.req.Header == nil {
		t.Fatalf("expected non-nil request Header")
	}
	if tr.req.Close {
		t.Error("got Close true, want false")
	}
	if g, e := tr.req.ContentLength, int64(len(json)); g != e {
		t.Errorf("got ContentLength %d, want %d", g, e)
	}
}

func TestPostFormRequestFormat(t *testing.T) {
	tr := &recordingTransport{}
	client := &Client{Transport: tr}

	urlStr := "http://dummy.faketld/"
	form := make(url.Values)
	form.Set("foo", "bar")
	form.Add("foo", "bar2")
	form.Set("bar", "baz")
	client.PostForm(urlStr, form) // Note: doesn't hit network

	if tr.req.Method != "POST" {
		t.Errorf("got method %q, want %q", tr.req.Method, "POST")
	}
	if tr.req.URL.String() != urlStr {
		t.Errorf("got URL %q, want %q", tr.req.URL.String(), urlStr)
	}
	if tr.req.Header == nil {
		t.Fatalf("expected non-nil request Header")
	}
	if g, e := tr.req.Header.Get("Content-Type"), "application/x-www-form-urlencoded"; g != e {
		t.Errorf("got Content-Type %q, want %q", g, e)
	}
	if tr.req.Close {
		t.Error("got Close true, want false")
	}
	// Depending on map iteration, body can be either of these.
	expectedBody := "foo=bar&foo=bar2&bar=baz"
	expectedBody1 := "bar=baz&foo=bar&foo=bar2"
	if g, e := tr.req.ContentLength, int64(len(expectedBody)); g != e {
		t.Errorf("got ContentLength %d, want %d", g, e)
	}
	bodyb, err := ioutil.ReadAll(tr.req.Body)
	if err != nil {
		t.Fatalf("ReadAll on req.Body: %v", err)
	}
	if g := string(bodyb); g != expectedBody && g != expectedBody1 {
		t.Errorf("got body %q, want %q or %q", g, expectedBody, expectedBody1)
	}
}

func TestRedirects(t *testing.T) {
	var ts *httptest.Server
	ts = httptest.NewServer(HandlerFunc(func(w ResponseWriter, r *Request) {
		n, _ := strconv.Atoi(r.FormValue("n"))
		// Test Referer header. (7 is arbitrary position to test at)
		if n == 7 {
			if g, e := r.Referer(), ts.URL+"/?n=6"; e != g {
				t.Errorf("on request ?n=7, expected referer of %q; got %q", e, g)
			}
		}
		if n < 15 {
			Redirect(w, r, fmt.Sprintf("/?n=%d", n+1), StatusFound)
			return
		}
		fmt.Fprintf(w, "n=%d", n)
	}))
	defer ts.Close()

	c := &Client{}
	_, err := c.Get(ts.URL)
	if e, g := "Get /?n=10: stopped after 10 redirects", fmt.Sprintf("%v", err); e != g {
		t.Errorf("with default client Get, expected error %q, got %q", e, g)
	}

	// HEAD request should also have the ability to follow redirects.
	_, err = c.Head(ts.URL)
	if e, g := "Head /?n=10: stopped after 10 redirects", fmt.Sprintf("%v", err); e != g {
		t.Errorf("with default client Head, expected error %q, got %q", e, g)
	}

	// Do should also follow redirects.
	greq, _ := NewRequest("GET", ts.URL, nil)
	_, err = c.Do(greq)
	if e, g := "Get /?n=10: stopped after 10 redirects", fmt.Sprintf("%v", err); e != g {
		t.Errorf("with default client Do, expected error %q, got %q", e, g)
	}

	var checkErr error
	var lastVia []*Request
	c = &Client{CheckRedirect: func(_ *Request, via []*Request) error {
		lastVia = via
		return checkErr
	}}
	res, err := c.Get(ts.URL)
	finalUrl := res.Request.URL.String()
	if e, g := "<nil>", fmt.Sprintf("%v", err); e != g {
		t.Errorf("with custom client, expected error %q, got %q", e, g)
	}
	if !strings.HasSuffix(finalUrl, "/?n=15") {
		t.Errorf("expected final url to end in /?n=15; got url %q", finalUrl)
	}
	if e, g := 15, len(lastVia); e != g {
		t.Errorf("expected lastVia to have contained %d elements; got %d", e, g)
	}

	checkErr = errors.New("no redirects allowed")
	res, err = c.Get(ts.URL)
	if urlError, ok := err.(*url.Error); !ok || urlError.Err != checkErr {
		t.Errorf("with redirects forbidden, expected a *url.Error with our 'no redirects allowed' error inside; got %#v (%q)", err, err)
	}
	if res == nil {
		t.Fatalf("Expected a non-nil Response on CheckRedirect failure (http://golang.org/issue/3795)")
	}
	if res.Header.Get("Location") == "" {
		t.Errorf("no Location header in Response")
	}
}

var expectedCookies = []*Cookie{
	{Name: "ChocolateChip", Value: "tasty"},
	{Name: "First", Value: "Hit"},
	{Name: "Second", Value: "Hit"},
}

var echoCookiesRedirectHandler = HandlerFunc(func(w ResponseWriter, r *Request) {
	for _, cookie := range r.Cookies() {
		SetCookie(w, cookie)
	}
	if r.URL.Path == "/" {
		SetCookie(w, expectedCookies[1])
		Redirect(w, r, "/second", StatusMovedPermanently)
	} else {
		SetCookie(w, expectedCookies[2])
		w.Write([]byte("hello"))
	}
})

func TestClientSendsCookieFromJar(t *testing.T) {
	tr := &recordingTransport{}
	client := &Client{Transport: tr}
	client.Jar = &TestJar{perURL: make(map[string][]*Cookie)}
	us := "http://dummy.faketld/"
	u, _ := url.Parse(us)
	client.Jar.SetCookies(u, expectedCookies)

	client.Get(us) // Note: doesn't hit network
	matchReturnedCookies(t, expectedCookies, tr.req.Cookies())

	client.Head(us) // Note: doesn't hit network
	matchReturnedCookies(t, expectedCookies, tr.req.Cookies())

	client.Post(us, "text/plain", strings.NewReader("body")) // Note: doesn't hit network
	matchReturnedCookies(t, expectedCookies, tr.req.Cookies())

	client.PostForm(us, url.Values{}) // Note: doesn't hit network
	matchReturnedCookies(t, expectedCookies, tr.req.Cookies())

	req, _ := NewRequest("GET", us, nil)
	client.Do(req) // Note: doesn't hit network
	matchReturnedCookies(t, expectedCookies, tr.req.Cookies())
}

// Just enough correctness for our redirect tests. Uses the URL.Host as the
// scope of all cookies.
type TestJar struct {
	m      sync.Mutex
	perURL map[string][]*Cookie
}

func (j *TestJar) SetCookies(u *url.URL, cookies []*Cookie) {
	j.m.Lock()
	defer j.m.Unlock()
	j.perURL[u.Host] = cookies
}

func (j *TestJar) Cookies(u *url.URL) []*Cookie {
	j.m.Lock()
	defer j.m.Unlock()
	return j.perURL[u.Host]
}

func TestRedirectCookiesOnRequest(t *testing.T) {
	var ts *httptest.Server
	ts = httptest.NewServer(echoCookiesRedirectHandler)
	defer ts.Close()
	c := &Client{}
	req, _ := NewRequest("GET", ts.URL, nil)
	req.AddCookie(expectedCookies[0])
	// TODO: Uncomment when an implementation of a RFC6265 cookie jar lands.
	_ = c
	// resp, _ := c.Do(req)
	// matchReturnedCookies(t, expectedCookies, resp.Cookies())

	req, _ = NewRequest("GET", ts.URL, nil)
	// resp, _ = c.Do(req)
	// matchReturnedCookies(t, expectedCookies[1:], resp.Cookies())
}

func TestRedirectCookiesJar(t *testing.T) {
	var ts *httptest.Server
	ts = httptest.NewServer(echoCookiesRedirectHandler)
	defer ts.Close()
	c := &Client{}
	c.Jar = &TestJar{perURL: make(map[string][]*Cookie)}
	u, _ := url.Parse(ts.URL)
	c.Jar.SetCookies(u, []*Cookie{expectedCookies[0]})
	resp, _ := c.Get(ts.URL)
	matchReturnedCookies(t, expectedCookies, resp.Cookies())
}

func matchReturnedCookies(t *testing.T, expected, given []*Cookie) {
	t.Logf("Received cookies: %v", given)
	if len(given) != len(expected) {
		t.Errorf("Expected %d cookies, got %d", len(expected), len(given))
	}
	for _, ec := range expected {
		foundC := false
		for _, c := range given {
			if ec.Name == c.Name && ec.Value == c.Value {
				foundC = true
				break
			}
		}
		if !foundC {
			t.Errorf("Missing cookie %v", ec)
		}
	}
}

func TestStreamingGet(t *testing.T) {
	say := make(chan string)
	ts := httptest.NewServer(HandlerFunc(func(w ResponseWriter, r *Request) {
		w.(Flusher).Flush()
		for str := range say {
			w.Write([]byte(str))
			w.(Flusher).Flush()
		}
	}))
	defer ts.Close()

	c := &Client{}
	res, err := c.Get(ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	var buf [10]byte
	for _, str := range []string{"i", "am", "also", "known", "as", "comet"} {
		say <- str
		n, err := io.ReadFull(res.Body, buf[0:len(str)])
		if err != nil {
			t.Fatalf("ReadFull on %q: %v", str, err)
		}
		if n != len(str) {
			t.Fatalf("Receiving %q, only read %d bytes", str, n)
		}
		got := string(buf[0:n])
		if got != str {
			t.Fatalf("Expected %q, got %q", str, got)
		}
	}
	close(say)
	_, err = io.ReadFull(res.Body, buf[0:1])
	if err != io.EOF {
		t.Fatalf("at end expected EOF, got %v", err)
	}
}

type writeCountingConn struct {
	net.Conn
	count *int
}

func (c *writeCountingConn) Write(p []byte) (int, error) {
	*c.count++
	return c.Conn.Write(p)
}

// TestClientWrites verifies that client requests are buffered and we
// don't send a TCP packet per line of the http request + body.
func TestClientWrites(t *testing.T) {
	ts := httptest.NewServer(HandlerFunc(func(w ResponseWriter, r *Request) {
	}))
	defer ts.Close()

	writes := 0
	dialer := func(netz string, addr string) (net.Conn, error) {
		c, err := net.Dial(netz, addr)
		if err == nil {
			c = &writeCountingConn{c, &writes}
		}
		return c, err
	}
	c := &Client{Transport: &Transport{Dial: dialer}}

	_, err := c.Get(ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	if writes != 1 {
		t.Errorf("Get request did %d Write calls, want 1", writes)
	}

	writes = 0
	_, err = c.PostForm(ts.URL, url.Values{"foo": {"bar"}})
	if err != nil {
		t.Fatal(err)
	}
	if writes != 1 {
		t.Errorf("Post request did %d Write calls, want 1", writes)
	}
}

func TestClientInsecureTransport(t *testing.T) {
	ts := httptest.NewTLSServer(HandlerFunc(func(w ResponseWriter, r *Request) {
		w.Write([]byte("Hello"))
	}))
	defer ts.Close()

	// TODO(bradfitz): add tests for skipping hostname checks too?
	// would require a new cert for testing, and probably
	// redundant with these tests.
	for _, insecure := range []bool{true, false} {
		tr := &Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: insecure,
			},
		}
		c := &Client{Transport: tr}
		_, err := c.Get(ts.URL)
		if (err == nil) != insecure {
			t.Errorf("insecure=%v: got unexpected err=%v", insecure, err)
		}
	}
}

func TestClientErrorWithRequestURI(t *testing.T) {
	req, _ := NewRequest("GET", "http://localhost:1234/", nil)
	req.RequestURI = "/this/field/is/illegal/and/should/error/"
	_, err := DefaultClient.Do(req)
	if err == nil {
		t.Fatalf("expected an error")
	}
	if !strings.Contains(err.Error(), "RequestURI") {
		t.Errorf("wanted error mentioning RequestURI; got error: %v", err)
	}
}
