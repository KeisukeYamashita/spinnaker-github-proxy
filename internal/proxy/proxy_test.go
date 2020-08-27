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

func getTestUserInfo(t *testing.T, login string) *github.UserInfo {
	t.Helper()
	return &github.UserInfo{
		Login: login,
	}
}

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
		getUserInfo            *github.UserInfo
		failGetUserInfo        bool
	}{
		"ok": {
			http.StatusOK,
			true,
			testBearer + "token",
			testAllowedOrg,
			[]github.Organization{{testAllowedOrg}, {"keke-test"}},
			false,
			getTestUserInfo(t, "KeisukeYamashita"),
			false,
		},
		"ok bypass": {
			http.StatusOK,
			true,
			testBearer + "token",
			"",
			[]github.Organization{{testAllowedOrg}, {"keke-test"}},
			false,
			getTestUserInfo(t, "KeisukeYamashita"),
			false,
		},
		"not belonging to org": {
			http.StatusForbidden,
			true,
			testBearer + "token",
			testAllowedOrg,
			[]github.Organization{{"keke-test"}},
			false,
			getTestUserInfo(t, "KeisukeYamashita"),
			false,
		},
		"empty authorization token": {
			http.StatusBadRequest,
			false,
			testBearer,
			testAllowedOrg,
			[]github.Organization{{"keke-test"}},
			false,
			getTestUserInfo(t, "KeisukeYamashita"),
			false,
		},
		"missing authorization token": {
			http.StatusBadRequest,
			false,
			testBearer,
			testAllowedOrg,
			[]github.Organization{{"keke-test"}},
			false,
			getTestUserInfo(t, "KeisukeYamashita"),
			false,
		},
		"wrong format authorization token": {
			http.StatusBadRequest,
			true,
			testBearer + "WRONG TOKEN FORMAT",
			testAllowedOrg,
			[]github.Organization{{"keke-test"}},
			false,
			getTestUserInfo(t, "KeisukeYamashita"),
			false,
		},
		"wrong token type": {
			http.StatusBadRequest,
			true,
			"oauth TOKEN",
			testAllowedOrg,
			[]github.Organization{{"keke-test"}},
			false,
			getTestUserInfo(t, "KeisukeYamashita"),
			false,
		},
		"failed to get user info": {
			http.StatusBadGateway,
			true,
			testBearer + "token",
			testAllowedOrg,
			[]github.Organization{{"keke-test"}},
			false,
			getTestUserInfo(t, "KeisukeYamashita"),
			true,
		},
		"failed to get orgs": {
			http.StatusBadGateway,
			true,
			testBearer + "token",
			testAllowedOrg,
			[]github.Organization{{"keke-test"}},
			true,
			getTestUserInfo(t, "KeisukeYamashita"),
			false,
		},
	}

	logger, _ := zap.NewDevelopment()
	for n, tc := range tcs {
		tc := tc
		t.Run(n, func(t *testing.T) {
			t.Parallel()

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
