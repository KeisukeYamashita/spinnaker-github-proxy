package proxy

import (
	"io"
	"net/http"
	"strings"

	"github.com/KeisukeYamashita/spinnaker-github-proxy/internal/github"
	"go.uber.org/zap"
)

var (
	errResps = map[int]string{
		http.StatusForbidden:  "authorization header missing",
		http.StatusBadGateway: "upstream server err",
		http.StatusBadRequest: "bad request",
	}
)

type Proxy interface {
	OAuthProxyHandler() http.Handler
}

type proxy struct {
	ghClient   github.Client
	logger     *zap.Logger
	allowedOrg string
}

// proxy implements Proxy interface
var _ Proxy = (*proxy)(nil)

type ProxyOption func(p *proxy)

func WithProxyLogger(l *zap.Logger) ProxyOption {
	return func(p *proxy) {
		p.logger = l
	}
}

func WithOrganizationRestriction(allowedOrg string) ProxyOption {
	return func(p *proxy) {
		p.allowedOrg = allowedOrg
	}
}

func NewProxyHandler(ghClient github.Client, opts ...ProxyOption) Proxy {
	p := &proxy{
		ghClient: ghClient,
	}

	for _, opt := range opts {
		opt(p)
	}

	if p.logger == nil {
		logger, _ := zap.NewProduction()
		p.logger = logger
	}

	return p
}

func (p *proxy) OAuthProxyHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p.oauthProxyHandler(w, r)
	})
}

func (p *proxy) oauthProxyHandler(w http.ResponseWriter, r *http.Request) {
	header := r.Header.Get("Authorization")
	if header == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(errResps[http.StatusBadRequest]))
		p.logger.Error("no authorization header")
		return
	}

	str := strings.Split(header, " ")
	if len(str) != 2 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(errResps[http.StatusBadRequest]))
		p.logger.Error("bad request")
		return
	}

	if str[1] == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(errResps[http.StatusBadRequest]))
		p.logger.Error("bad request, empty token")
		return
	}

	tokenType, token := str[0], str[1]
	if strings.ToLower(tokenType) != "bearer" {
		w.WriteHeader(http.StatusBadRequest)
		msg := "token type should be bearer type"
		w.Write([]byte(msg))
		p.logger.Error("msg", zap.String("tokenType", tokenType))
		return
	}

	userInfoResp, err := p.ghClient.GetUserInfo(token)
	if err != nil {
		msg := "error while getting user info"
		p.logger.Error(msg, zap.Error(err))
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte(msg))
		return
	}

	orgs, err := p.ghClient.GetOrgs(token)
	if err != nil {
		msg := "error while getting user's organization info"
		p.logger.Error(msg, zap.Error(err))
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte(msg))
		return
	}

	if p.allowedOrg != "" {
		if orgs.LoggedInto(p.allowedOrg) {
			p.logger.Info("organization belonging user", zap.String("allowedOrganization", p.allowedOrg), zap.Array("organizations", orgs))
			w.Header().Set("Content-Type", "application/json")
			io.Copy(w, userInfoResp.Body)
			return
		}

		msg := "user is not a member of allowed orgs"
		p.logger.Info(msg, zap.String("allowedOrganization", p.allowedOrg), zap.Array("organizations", orgs))
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(msg))
		return
	}

	msg := "user is allowed to bypass with any GitHub Organization"
	p.logger.Info(msg, zap.Array("organizations", orgs))
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(msg))
}
