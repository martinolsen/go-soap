package soap

import (
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type Envelope struct {
	XMLName xml.Name `xml:"http://www.w3.org/2003/05/soap-envelope Envelope"`
	Header  *Header  `xml:",omitempty"`
	Body    *Body    `xml:""`
}

func TestEnvelope(t *testing.T) {
	if bytes, err := xml.Marshal(&Envelope{}); err != nil {
		t.Fatalf("could not marshal envelope: %s", err)
	} else if s := string(bytes); s != `<Envelope xmlns="http://www.w3.org/2003/05/soap-envelope"></Envelope>` {
		t.Errorf("unexpected marshaling: `%s`", s)
	} else {
		t.Logf("OK - marshal empty envelope")
	}

	if bytes, err := xml.Marshal(&Envelope{Header: &Header{}}); err != nil {
		t.Fatalf("could not marshal envelope: %s", err)
	} else if s := string(bytes); s != `<Envelope xmlns="http://www.w3.org/2003/05/soap-envelope"></Envelope>` {
		t.Errorf("unexpected marshaling: `%s`", s)
	} else {
		t.Logf("OK - marshal empty envelope")
	}

}

type Header struct {
	XMLName     xml.Name     `xml:"http://www.w3.org/2003/05/soap-envelope Header"`
	HeaderItems []HeaderItem `xml:",omitempty"`
}

type HeaderItem struct {
	EncodingStyle  string `xml:"encodingStyle,attr,omitempty"`
	Role           string `xml:"role,attr,omitempty"`
	MustUnderstand string `xml:"mustUnderstand,attr,omitempty"`
	Relay          string `xml:"relay,attr,omitempty"`
}

type Body struct {
	XMLName xml.Name `xml:"http://www.w3.org/2003/05/soap-envelope Body"`
	Fault   *Fault   `xml:",omitempty"`
}

type Fault struct {
	XMLName xml.Name `xml:"http://www.w3.org/2003/05/soap-envelope Fault"`
	Code    *Code    `xml:""`
}

type Code struct {
	XMLName xml.Name `xml:"http://www.w3.org/2003/05/soap-envelope Code"`
	Value   string   `xml:""` // TODO - env:faultCodeEnum -> xs:QName
	Subcode *Subcode `xml:",omitempty"`
}

type Subcode struct {
	XMLName xml.Name     `xml:"http://www.w3.org/2003/05/soap-envelope Subcode"`
	Value   SubcodeValue `xml:""`
	Subcode *Subcode     `xml:",omitempty"`
}

type SubcodeValue string // TODO - xs:QName

type Client struct{}

func NewClient() *Client {
	return nil
}

func (c *Client) Call() {}

type testEndpoint struct{}

func (e *testEndpoint) Foo(a, b string, c int) { panic("todo - Foo") }

type handler struct{}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) { panic("todo - ServeHTTP") }

type Server struct {
	*httptest.Server
	mux *http.ServeMux
}

func (s *Server) Handle(v interface{}) {
	s.mux.HandleFunc("/", s.handler)
}

func (s *Server) handler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not acceptable", http.StatusMethodNotAllowed)
		return
	}

	if ct := r.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/soap+xml") {
		http.Error(w, "Content-Type not acceptable", http.StatusNotAcceptable)
		return
	}

	action := getAction(r)

	panic("todo - handler " + action)
}

// `getAction` returns the first of Content-Type headers 'action' parameter or
// SOAPAction header.
func getAction(r *http.Request) string {
	for _, v := range strings.Split(r.Header.Get("Content-Type"), ";") {
		vt := strings.TrimSpace(v)
		if strings.HasPrefix(vt, "action=") {
			return strings.TrimPrefix(vt, "action=")
		}
	}

	if soapAction := r.Header.Get("SOAPAction"); soapAction != "" {
		// TODO - handle URIs properly
		vs := strings.Split(soapAction, "/")
		return vs[len(vs)-1]
	}

	return ""
}

func TestGetAction(t *testing.T) {
	// No action provided
	r := &http.Request{
		Method: "POST",
	}

	if a := getAction(r); a != "" {
		t.Errorf("FAIL - no action")
	}

	// Header
	r.Header = http.Header{}
	r.Header.Set("SOAPAction", "http://example.org/foo-header")
	if getAction(r) != "foo-header" {
		t.Errorf("FAIL - header")
	}

	// Content Type
	r.Header.Set("Content-Type", "application/soap+xml; action=foo-ct")
	if getAction(r) != "foo-ct" {
		t.Errorf("FAIL - content type")
	}
}

func NewServer() *Server {
	mux := http.NewServeMux()

	s := &Server{httptest.NewServer(mux), mux}

	return s
}

func TestHandle(t *testing.T) {
	s := NewServer()
	defer s.Close()

	s.Handle(&testEndpoint{})

	if r, err := http.Get(s.URL); err != nil {
		t.Fatalf("FAIL - POST %q: %s", s.URL, err)
	} else if r.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("FAIL - status %q", r.Status)
	} else {
		t.Logf("OK - GET %q", s.URL)
	}

	ReqFoo := `<? xml version="1.0" encoding="utf-8">
	<e:Envelope xmlns:e="http://www.w3.org/2003/05/soap-envelope">
		<e:Body>
			<Foo>
				<a>alpha</a>
				<b>bandit</b>
				<c>42</c>
			</Foo>
		</e:Body>
	</e:Envelope>`

	if r, err := http.Post(s.URL, "", strings.NewReader(ReqFoo)); err != nil {
		t.Fatalf("FAIL - POST %q: %s", s.URL, err)
	} else if r.StatusCode != http.StatusNotAcceptable {
		t.Errorf("FAIL - status %q", r.Status)
	} else {
		t.Logf("OK - POST %q", s.URL)
	}

	if r, err := http.Post(s.URL, "application/soap+xml; charset=utf-8; action=Foo", strings.NewReader(ReqFoo)); err != nil {
		t.Fatalf("FAIL - POST %q: %s", s.URL, err)
	} else if r.StatusCode != http.StatusOK {
		t.Errorf("FAIL - status %q", r.Status)
	} else {
		t.Logf("OK - POST %q", s.URL)
	}
}
