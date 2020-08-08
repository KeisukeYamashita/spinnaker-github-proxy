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
)

func TestProxy_OAuthProxyHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tcs := map[string]struct {
		wantStatusCode         int
		withAuthorazationToken bool
		getOrgs                github.Organizations
		getUserInfo            *http.Response
	}{
		"ok": {
			http.StatusOK,
			true,
			[]github.Organization{{testAllowedOrg}, {"keke-test"}},
			&http.Response{Body: &http.NoBody},
		},
		"not belonging to org": {
			http.StatusForbidden,
			true,
			[]github.Organization{{"keke-test"}},
			&http.Response{Body: &http.NoBody},
		},
		"empty authorization token": {
			http.StatusBadRequest,
			false,
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
			proxy := proxy{
				ghClient:   ghClientMock,
				allowedOrg: testAllowedOrg,
				logger:     logger,
			}

			req, err := http.NewRequest("POST", "/", nil)
			if tc.withAuthorazationToken {
				req.Header.Add("Authorization", "Bearer TOKEN")
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
