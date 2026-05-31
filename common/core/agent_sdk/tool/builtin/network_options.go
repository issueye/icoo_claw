package toolbuiltin

import (
	"net/http"
	"net/url"
	"strings"
)

// NetworkOptions configures outbound HTTP clients used by network tools.
type NetworkOptions struct {
	HTTPProxy  string
	HTTPSProxy string
	NoProxy    string
}

func (o NetworkOptions) transport() (*http.Transport, error) {
	proxyFunc, err := o.proxyFunc()
	if err != nil {
		return nil, err
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = proxyFunc
	return transport, nil
}

func (o NetworkOptions) proxyFunc() (func(*http.Request) (*url.URL, error), error) {
	httpProxy := strings.TrimSpace(o.HTTPProxy)
	httpsProxy := strings.TrimSpace(o.HTTPSProxy)
	if httpProxy == "" && httpsProxy == "" && strings.TrimSpace(o.NoProxy) == "" {
		return http.ProxyFromEnvironment, nil
	}

	var httpURL *url.URL
	var httpsURL *url.URL
	var err error
	if httpProxy != "" {
		httpURL, err = parseProxyURL(httpProxy)
		if err != nil {
			return nil, err
		}
	}
	if httpsProxy != "" {
		httpsURL, err = parseProxyURL(httpsProxy)
		if err != nil {
			return nil, err
		}
	}
	noProxy := parseNoProxy(o.NoProxy)
	return func(req *http.Request) (*url.URL, error) {
		if req == nil || req.URL == nil {
			return nil, nil
		}
		if noProxy.matches(req.URL.Hostname()) {
			return nil, nil
		}
		switch strings.ToLower(req.URL.Scheme) {
		case "https":
			if httpsURL != nil {
				return httpsURL, nil
			}
			return httpURL, nil
		case "http":
			if httpURL != nil {
				return httpURL, nil
			}
			return httpsURL, nil
		default:
			return nil, nil
		}
	}, nil
}

func parseProxyURL(value string) (*url.URL, error) {
	if !strings.Contains(value, "://") {
		value = "http://" + value
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return nil, err
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return nil, url.InvalidHostError(value)
	}
	return parsed, nil
}

type noProxyList []string

func parseNoProxy(value string) noProxyList {
	var out []string
	for _, item := range strings.Split(value, ",") {
		item = strings.ToLower(strings.TrimSpace(item))
		if item != "" {
			out = append(out, item)
		}
	}
	return noProxyList(out)
}

func (n noProxyList) matches(host string) bool {
	host = strings.ToLower(strings.Trim(host, "[]"))
	if host == "" {
		return false
	}
	for _, item := range n {
		switch {
		case item == "*":
			return true
		case item == host:
			return true
		case strings.HasPrefix(item, ".") && strings.HasSuffix(host, item):
			return true
		case strings.HasSuffix(host, "."+item):
			return true
		}
	}
	return false
}
