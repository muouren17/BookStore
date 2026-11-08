package discovery

import (
	"fmt"

	"google.golang.org/grpc/resolver"
)

const (
	scheme      = "static"
	serviceName = "bookstore"
)

var backends = map[string][]string{
	serviceName: {"127.0.0.1:8090"},
}

type Resolver struct {
	target   resolver.Target
	cc       resolver.ClientConn
	backends map[string][]string
}

func (r *Resolver) ResolveNow(_ resolver.ResolveNowOptions) {
	name := r.target.Endpoint()
	addresses := r.backends[name]
	if len(addresses) == 0 {
		r.cc.ReportError(fmt.Errorf("找不到服务 %q 的地址", name))
		return
	}
	resolved := make([]resolver.Address, len(addresses))
	for i, address := range addresses {
		resolved[i] = resolver.Address{
			Addr: address,
		}
	}
	err := r.cc.UpdateState(resolver.State{
		Addresses: resolved,
	})
	if err != nil {
		r.cc.ReportError(fmt.Errorf("更新服务 %q 的解析结果失败: %w", name, err))
	}

}

func (*Resolver) Close() {}

type Builder struct{}

func (*Builder) Scheme() string {
	return scheme
}

func (*Builder) Build(
	target resolver.Target,
	cc resolver.ClientConn,
	_ resolver.BuildOptions,
) (resolver.Resolver, error) {
	r := &Resolver{
		target:   target,
		cc:       cc,
		backends: backends,
	}
	r.ResolveNow(resolver.ResolveNowOptions{})
	return r, nil
}
