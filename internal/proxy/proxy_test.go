package proxy

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"

	"github.com/KeisukeYamashita/spinnaker-github-proxy/internal/github"
	"github.com/golang/mock/gomock"
)

const (
	testAllowedOrg = "testOrg"
	testBearer     = "Bearer "
)

func TestProxy_OAuthProxyHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tcs := map[string]struct {
		wantStatusCode         int
		withAuthorizationToken bool
		token                  string
		allowedOrg             string
		getOrgs                github.Organizations
		getUserInfo            *http.Response
	}{
		"ok": {
			http.StatusOK,
			true,
			testBearer + "token",
			testAllowedOrg,
			[]github.Organization{{testAllowedOrg}, {"keke-test"}},
			&http.Response{Body: &http.NoBody},
		},
		"not belonging to org": {
			http.StatusForbidden,
			true,
			testBearer + "token",
			testAllowedOrg,
			[]github.Organization{{"keke-test"}},
			&http.Response{Body: &http.NoBody},
		},
		"empty authorization token": {
			http.StatusBadRequest,
			false,
			testBearer + "token",
			testAllowedOrg,
			[]github.Organization{{"keke-test"}},
			&http.Response{Body: &http.NoBody},
		},
		"missing authorization token": {
			http.StatusBadRequest,
			false,
			testBearer,
			testAllowedOrg,
			[]github.Organization{{"keke-test"}},
			&http.Response{Body: &http.NoBody},
		},
		"wrong format authorization token": {
			http.StatusBadRequest,
			false,
			testAllowedOrg,
			testBearer + "WRONG TOKEN FORMAT",
			[]github.Organization{{"keke-test"}},
			&http.Response{Body: &http.NoBody},
		},
		"wrong token type": {
			http.StatusBadRequest,
			false,
			"oauth",
			testAllowedOrg,
			[]github.Organization{{"keke-test"}},
			&http.Response{Body: &http.NoBody},
		},
	}

	logger, _ := zap.NewDevelopment()
	for n, tc := range tcs {
		tc := tc
		t.Run(n, func(t *testing.T) {
			ghClientMock := github.NewMockClient(ctrl)
			ghClientMock.EXPECT().GetUserInfo(gomock.Any()).Return(tc.getUserInfo, nil).AnyTimes()
			ghClientMock.EXPECT().GetOrgs(gomock.Any()).Return(tc.getOrgs, nil).AnyTimes()
			proxy := NewProxyHandler(ghClientMock, WithOrganizationRestriction(tc.allowedOrg), WithProxyLogger(logger))

			req, err := http.NewRequest("POST", "/", nil)
			if tc.withAuthorizationToken {
				req.Header.Add("Authorization", tc.token)
			}

			if err != nil {
				t.Errorf("failed to create req: %v", err)
			}

			rr := httptest.NewRecorder()
			h := proxy.OAuthProxyHandler()
			h.ServeHTTP(rr, req)

			if status := rr.Code; status != tc.wantStatusCode {
				t.Errorf("unexpected status code got: %d, want: %d", rr.Code, tc.wantStatusCode)
			}
		})
	}
}
