package portal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/labstack/echo"
	"github.com/stretchr/testify/assert"
)

var testRPCURL *url.URL

func init() {
	// recryptRPC, found := os.LookupEnv("RECRYPT_RPC")
	// if !found {
	// 	fmt.Println("Please specify RPC url with RECRYPT_RPC environment variable")
	// 	os.Exit(1)
	// }

	// recryptRPCURL, err := url.Parse(recryptRPC)
	recryptRPCURL, err := url.Parse("http://howard:yeh@localhost:18489")
	if err != nil {
		log.Println("Invalid RECRYPT_RPC", recryptRPCURL.String())
	}

	testRPCURL = recryptRPCURL
}

func testServer() *Server {
	opts := ServerOption{
		// Port:        9999,
		RecryptdRPCURL: testRPCURL,
	}

	s := NewServer(opts)

	return s
}

func testReq(handler func(c echo.Context) error, req *http.Request) (*http.Response, error) {
	e := echo.New()
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler(c)

	return rec.Result(), err
}

func testProxy(s *Server, jsonreq *jsonRPCRequest) (*http.Response, error) {
	jsonreqBodyBytes, err := json.Marshal(jsonreq)
	if err != nil {
		return nil, err
	}

	jsonreqBody := bytes.NewReader(jsonreqBodyBytes)
	req := httptest.NewRequest("POST", "/", jsonreqBody)
	req.Header.Set("Content-Type", "application/json")

	res, err := testReq(s.proxyRPC, req)

	return res, nil
}

func TestProxyMethodNotFound(t *testing.T) {
	is := assert.New(t)

	s := testServer()

	jsonreq := &jsonRPCRequest{
		Method: "no-such-method",
	}

	res, err := testProxy(s, jsonreq)
	is.NoError(err)
	is.Equal(404, res.StatusCode)
}

func TestProxyMethodFound(t *testing.T) {
	is := assert.New(t)

	s := testServer()

	jsonreq := &jsonRPCRequest{
		Method: "getinfo",
	}

	res, err := testProxy(s, jsonreq)
	is.NoError(err)
	is.Equal(200, res.StatusCode)
}

func TestProxyUserAuthorization(t *testing.T) {
	is := assert.New(t)
	s := testServer()

	hasNumberOfPendingAuths := func(i int) {
		var auths []*Authorization
		listAuthsReq := httptest.NewRequest("GET", "/authorizations", nil)
		listAuthsRes, err := testReq(s.listAuthorizations, listAuthsReq)
		is.NoError(err)
		defer listAuthsRes.Body.Close()
		is.Equal(http.StatusOK, listAuthsRes.StatusCode)

		dec := json.NewDecoder(listAuthsRes.Body)
		err = dec.Decode(&auths)
		is.NoError(err)
		is.Equal(i, len(auths))
	}

	makeAuthCall := func() *Authorization {
		jsonreq := &jsonRPCRequest{
			Method: "getnewaddress",
		}

		res, err := testProxy(s, jsonreq)
		is.NoError(err)
		is.Equal(402, res.StatusCode)
		defer res.Body.Close()

		var auth Authorization
		dec := json.NewDecoder(res.Body)
		err = dec.Decode(&auth)
		is.NoError(err)

		return &auth
	}

	getAuth := func(id string) *Authorization {
		url := fmt.Sprintf("/authorizations/%s", id)
		req := httptest.NewRequest("GET", url, nil)
		rec := httptest.NewRecorder()
		s.authApp.ServeHTTP(rec, req)
		res := rec.Result()

		defer res.Body.Close()

		// io.Copy(os.Stdout, res.Body)

		var auth Authorization
		dec := json.NewDecoder(res.Body)
		err := dec.Decode(&auth)
		is.NoError(err)

		return &auth
	}

	hasNumberOfPendingAuths(0)

	auth1 := makeAuthCall()
	hasNumberOfPendingAuths(1)

	auth2 := makeAuthCall()
	hasNumberOfPendingAuths(2)

	is.NotEqual(auth1.ID, auth2.ID)

	// Accept an authorization
	is.Equal(auth1.State, AuthorizationPending)
	req := httptest.NewRequest("POST", fmt.Sprintf("/authorizations/%s/accept", auth1.ID), nil)
	rec := httptest.NewRecorder()
	s.authApp.ServeHTTP(rec, req)
	res := rec.Result()
	is.Equal(http.StatusOK, res.StatusCode)
	auth1 = getAuth(auth1.ID)
	is.Equal(AuthorizationAccepted, auth1.State)

	// Re-execute an authorization
	// jsonreq := &jsonRPCRequest{
	// 	Method: "getnewaddress",
	// }
	req = httptest.NewRequest("POST", "/", bytes.NewReader([]byte(
		fmt.Sprintf(`{"method": "getnewaddress", "auth": "%s", "params": null}`, auth1.ID),
	)))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	s.proxyApp.ServeHTTP(rec, req)
	res = rec.Result()
	is.Equal(http.StatusOK, res.StatusCode)

	// Reusing auth token should return 403
	req = httptest.NewRequest("POST", "/", bytes.NewReader([]byte(
		fmt.Sprintf(`{"method": "getnewaddress", "auth": "%s", "params": null}`, auth1.ID),
	)))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	s.proxyApp.ServeHTTP(rec, req)
	res = rec.Result()
	is.Equal(http.StatusForbidden, res.StatusCode)
}
