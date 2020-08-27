package proxy

import (
	"encoding/json"
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
		p.logger.Error("no authorization header")
		http.Error(w, "no authorization header", http.StatusBadRequest)
		return
	}

	str := strings.Split(header, " ")
	if len(str) != 2 {
		p.logger.Error("bad request")
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	tokenType, token := str[0], str[1]
	if strings.ToLower(tokenType) != "bearer" {
		msg := "token type should be bearer type"
		p.logger.Error("msg", zap.String("tokenType", tokenType))
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	u, err := p.ghClient.GetUserInfo(token)
	if err != nil {
		msg := "error while getting user info"
		p.logger.Error(msg, zap.Error(err))
		http.Error(w, msg, http.StatusBadGateway)
		return
	}

	orgs, err := p.ghClient.GetOrgs(token)
	if err != nil {
		msg := "error while getting user's organization info"
		p.logger.Error(msg, zap.Error(err))
		http.Error(w, msg, http.StatusBadGateway)
		return
	}

	if p.allowedOrg != "" {
		if orgs.LoggedInto(p.allowedOrg) {
			p.logger.Info("organization belonging user", zap.String("allowedOrganization", p.allowedOrg), zap.Array("organizations", orgs), zap.String("user", u.Login))
			w.Header().Set("Content-Type", "application/json")
			b, err := json.Marshal(u)
			if err != nil {
				msg := "failed to marshal body"
				http.Error(w, msg, http.StatusInternalServerError)
				return
			}
			w.Write(b)
			return
		}

		msg := "user is not a member of allowed orgs"
		p.logger.Info(msg, zap.String("allowedOrganization", p.allowedOrg), zap.Array("organizations", orgs), zap.String("user", u.Login))
		http.Error(w, msg, http.StatusForbidden)
		return
	}

	msg := "user is allowed to bypass with any GitHub Organization"
	p.logger.Info(msg, zap.Array("organizations", orgs), zap.String("user", u.Login))
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(msg))
}
