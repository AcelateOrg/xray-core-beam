package session

import "context"

// RouteInfo is the routing outcome of one dispatched flow (Beam).
type RouteInfo struct {
	// OutboundTag is the outbound handler that carries the flow.
	OutboundTag string
	// RuleTag is the ruleTag of the routing rule that matched, "" when the
	// default handler was taken or the outbound was forced.
	RuleTag string
	// Domain is the sniffed (or requested) domain, "" when only an IP is
	// known.
	Domain string
	// Protocol is the sniffed protocol ("tls", "http1", "quic", ...).
	Protocol string
}

// RouteObserver receives the routing outcome of a flow. It is called
// synchronously on the dispatch path and must not block.
type RouteObserver func(RouteInfo)

type routeObserverKey struct{}

// ContextWithRouteObserver attaches an observer that the dispatcher calls
// once the outbound of a flow is chosen.
func ContextWithRouteObserver(ctx context.Context, observer RouteObserver) context.Context {
	return context.WithValue(ctx, routeObserverKey{}, observer)
}

// RouteObserverFromContext returns the attached observer, nil if none.
func RouteObserverFromContext(ctx context.Context) RouteObserver {
	observer, _ := ctx.Value(routeObserverKey{}).(RouteObserver)
	return observer
}
