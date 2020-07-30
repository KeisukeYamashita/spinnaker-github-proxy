package github

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWithBaseURL(t *testing.T) {
	tcs := map[string]struct {
		want string
	}{
		"ok":    {"https://keisukeyamashita.com"},
		"blank": {""},
	}

	for n, tc := range tcs {
		tc := tc
		t.Run(n, func(t *testing.T) {
			t.Parallel()
			c := &client{}
			opt := WithBaseURL(tc.want)
			opt(c)

			if c.baseURL != tc.want {
				t.Errorf("Unexpected return got %s want %s", c.baseURL, tc.want)
			}
		})
	}
}

func TestClient_GetUserInfo(t *testing.T) {
	tcs := map[string]struct {
		token string
		pass  bool
	}{
		"ok":          {"token", true},
		"empty token": {"", false},
	}

	for n, tc := range tcs {
		tc := tc
		t.Run(n, func(t *testing.T) {
			testserver := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				header := req.Header.Get("Authorization")
				if header == "" {
					rw.WriteHeader(http.StatusBadRequest)
					return
				}

				rw.Write([]byte(`OK`))
			}))
			defer testserver.Close()

			c := &client{
				client:  testserver.Client(),
				baseURL: testserver.URL,
			}

			_, err := c.GetOrgs(tc.token)
			if err != nil {
				if tc.pass {
					t.Errorf("failed: %v", err)
				}
			}
		})
	}
}

func TestOrganizations_LoggedInto(t *testing.T) {
	tcs := map[string]struct {
		orgs       Organizations
		allowedOrg string
		want       bool
	}{
		"ok":                             {[]Organization{{"keke"}, {"laboratory"}, {"world"}}, "keke", true},
		"not allowed org":                {[]Organization{{"keke"}, {"laboratory"}, {"world"}}, "not-allowed", false},
		"no organization belonging user": {[]Organization{}, "keke", false},
		"no allowed org":                 {[]Organization{{"keke"}, {"laboratory"}, {"world"}}, "", false},
		"no organization belonging user and no allowed org": {[]Organization{}, "", false},
	}

	for n, tc := range tcs {
		tc := tc
		t.Run(n, func(t *testing.T) {
			t.Parallel()

			if tc.orgs.LoggedInto(tc.allowedOrg) != tc.want {
				t.Errorf("failed to validate login")
			}
		})
	}
}
