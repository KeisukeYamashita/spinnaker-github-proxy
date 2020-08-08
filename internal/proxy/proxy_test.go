package proxy

import (
	"errors"
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
		failGetOrgs            bool
		getUserInfo            *http.Response
		failGetUserInfo        bool
	}{
		"ok": {
			http.StatusOK,
			true,
			testBearer + "token",
			testAllowedOrg,
			[]github.Organization{{testAllowedOrg}, {"keke-test"}},
			false,
			&http.Response{Body: &http.NoBody},
			false,
		},
		"ok bypass": {
			http.StatusOK,
			true,
			testBearer + "token",
			"",
			[]github.Organization{{testAllowedOrg}, {"keke-test"}},
			false,
			&http.Response{Body: &http.NoBody},
			false,
		},
		"not belonging to org": {
			http.StatusForbidden,
			true,
			testBearer + "token",
			testAllowedOrg,
			[]github.Organization{{"keke-test"}},
			false,
			&http.Response{Body: &http.NoBody},
			false,
		},
		"empty authorization token": {
			http.StatusBadRequest,
			false,
			testBearer,
			testAllowedOrg,
			[]github.Organization{{"keke-test"}},
			false,
			&http.Response{Body: &http.NoBody},
			false,
		},
		"missing authorization token": {
			http.StatusBadRequest,
			false,
			testBearer,
			testAllowedOrg,
			[]github.Organization{{"keke-test"}},
			false,
			&http.Response{Body: &http.NoBody},
			false,
		},
		"wrong format authorization token": {
			http.StatusBadRequest,
			true,
			testBearer + "WRONG TOKEN FORMAT",
			testAllowedOrg,
			[]github.Organization{{"keke-test"}},
			false,
			&http.Response{Body: &http.NoBody},
			false,
		},
		"wrong token type": {
			http.StatusBadRequest,
			true,
			"oauth TOKEN",
			testAllowedOrg,
			[]github.Organization{{"keke-test"}},
			false,
			&http.Response{Body: &http.NoBody},
			false,
		},
		"failed to get user info": {
			http.StatusBadGateway,
			true,
			testBearer + "token",
			testAllowedOrg,
			[]github.Organization{{"keke-test"}},
			false,
			&http.Response{Body: &http.NoBody},
			true,
		},
		"failed to get orgs": {
			http.StatusBadGateway,
			true,
			testBearer + "token",
			testAllowedOrg,
			[]github.Organization{{"keke-test"}},
			true,
			&http.Response{Body: &http.NoBody},
			false,
		},
	}

	logger, _ := zap.NewDevelopment()
	for n, tc := range tcs {
		tc := tc
		t.Run(n, func(t *testing.T) {
			ghClientMock := github.NewMockClient(ctrl)

			var errGetUserInfo error = nil
			var errGetOrg error = nil

			if tc.failGetOrgs {
				errGetOrg = errors.New("failed to get org")
			}

			if tc.failGetUserInfo {
				errGetUserInfo = errors.New("failed to get user info")
			}

			ghClientMock.EXPECT().GetUserInfo(gomock.Any()).Return(tc.getUserInfo, errGetUserInfo).AnyTimes()
			ghClientMock.EXPECT().GetOrgs(gomock.Any()).Return(tc.getOrgs, errGetOrg).AnyTimes()
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
