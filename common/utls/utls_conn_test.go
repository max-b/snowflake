package utls

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"testing"

	utls "github.com/refraction-networking/utls"
	. "github.com/smartystreets/goconvey/convey"
)

func newHTTPSServer(statusOverride int, body []byte) net.Listener {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(statusOverride)
		_, err := w.Write(body)
		if err != nil {
			So(err, ShouldBeNil)
		}
	})

	// l, err := net.Listen("tcp", "localhost:0")
	l, err := selfSignedTLSListen("tcp", "localhost:0")
	if err != nil {
		So(err, ShouldBeNil)
	}

	go http.ServeTLS(l, mux, "./testdata/cert.pem", "./testdata/key.pem")

	return l
}

func shouldEqualBytes(actual interface{}, expected ...interface{}) string {
	if bytes.Equal(actual.([]byte), expected[0].([]byte)) {
		return "" // empty string means the assertion passed
	}
	return fmt.Sprintf("actual bytes:\n%+v\ndid not match expected\n%+v", actual, expected)
}

func shouldContainBytes(actual interface{}, expected ...interface{}) string {
	if bytes.Contains(actual.([]byte), expected[0].([]byte)) {
		return "" // empty string means the assertion passed
	}
	return "actual bytes did not match expected"
}

func checkRoundTrips(t *testing.T, clientHelloID string, fetchURL string, expected []byte) {
	rt, err := NewUTLSRoundTripper(clientHelloID, &utls.Config{InsecureSkipVerify: true}, nil, nil)
	if err != nil {
		So(err, ShouldBeNil)
	}

	client := &http.Client{Transport: rt}
	res, err := client.Get(fetchURL)
	if err != nil {
		So(err, ShouldBeNil)
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		So(err, ShouldBeNil)
	}

	So(body, shouldEqualBytes, expected)
}

func TestRoundTripClientIDs(t *testing.T) {
	Convey("UTLS RoundTripper", t, func() {
		Convey("Successfully requests from TLS HTTP Server", func() {
			expected := "hello world"
			listener := newHTTPSServer(200, []byte(expected))
			defer listener.Close()

			clientHelloIDs := []string{
				"hellofirefox_auto",
				"hellofirefox_55",
				"hellofirefox_56",
				"hellofirefox_63",
				"hellofirefox_65",
				"hellochrome_auto",
				"hellochrome_58",
				"hellochrome_62",
				"hellochrome_70",
				"hellochrome_72",
				"helloios_auto",
				"helloios_11_1",
				"helloios_12_1",
			}

			fetchURL := url.URL{Scheme: "https", Host: listener.Addr().String()}
			fetchURLString := fmt.Sprintf("https://localhost:%s/", fetchURL.Port())
			for _, id := range clientHelloIDs {
				t.Logf("Checking ClientHelloID: %s", id)
				checkRoundTrips(t, id, fetchURLString, []byte(expected))
			}
		})
	})
}
