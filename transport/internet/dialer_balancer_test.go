package internet

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/xtls/xray-core/common"
	"github.com/xtls/xray-core/common/net"
	"github.com/xtls/xray-core/common/serial"
	"github.com/xtls/xray-core/common/session"
	"github.com/xtls/xray-core/features/outbound"
	routing_feature "github.com/xtls/xray-core/features/routing"
	"github.com/xtls/xray-core/transport"
)

type fakeOutboundHandler struct {
	tag        string
	dispatched chan []*session.Outbound
}

func (h *fakeOutboundHandler) Start() error { return nil }
func (h *fakeOutboundHandler) Close() error { return nil }
func (h *fakeOutboundHandler) Tag() string  { return h.tag }
func (h *fakeOutboundHandler) SenderSettings() *serial.TypedMessage {
	return nil
}
func (h *fakeOutboundHandler) ProxySettings() *serial.TypedMessage {
	return nil
}
func (h *fakeOutboundHandler) Dispatch(ctx context.Context, link *transport.Link) {
	select {
	case h.dispatched <- session.OutboundsFromContext(ctx):
	default:
	}
	_ = common.Close(link.Writer)
}

type fakeOutboundManager struct {
	handler outbound.Handler
}

func (m *fakeOutboundManager) Type() interface{} { return outbound.ManagerType() }
func (m *fakeOutboundManager) Start() error      { return nil }
func (m *fakeOutboundManager) Close() error      { return nil }
func (m *fakeOutboundManager) GetHandler(tag string) outbound.Handler {
	if m.handler != nil && m.handler.Tag() == tag {
		return m.handler
	}
	return nil
}
func (m *fakeOutboundManager) GetDefaultHandler() outbound.Handler { return nil }
func (m *fakeOutboundManager) AddHandler(context.Context, outbound.Handler) error {
	return nil
}
func (m *fakeOutboundManager) RemoveHandler(context.Context, string) error { return nil }
func (m *fakeOutboundManager) ListHandlers(context.Context) []outbound.Handler {
	return nil
}

type fakeBalancerPicker struct {
	tag      string
	resolved string
	err      error
	mu       sync.Mutex
	picked   []string
}

func (p *fakeBalancerPicker) PickBalancerOutbound(tag string) (string, error) {
	p.mu.Lock()
	p.picked = append(p.picked, tag)
	p.mu.Unlock()
	if p.err != nil {
		return "", p.err
	}
	if tag != p.tag {
		return "", context.Canceled
	}
	return p.resolved, nil
}

var (
	_ outbound.Manager               = (*fakeOutboundManager)(nil)
	_ routing_feature.BalancerPicker = (*fakeBalancerPicker)(nil)
)

func TestDialSystemUsesDialerBalancerTag(t *testing.T) {
	oldObm := obm
	oldPicker := balancerPicker
	t.Cleanup(func() {
		obm = oldObm
		balancerPicker = oldPicker
	})

	handler := &fakeOutboundHandler{
		tag:        "proxy-a",
		dispatched: make(chan []*session.Outbound, 1),
	}
	obm = &fakeOutboundManager{handler: handler}
	balancerPicker = &fakeBalancerPicker{tag: "bal", resolved: "proxy-a"}

	conn, err := DialSystem(context.Background(), net.TCPDestination(net.DomainAddress("example.com"), 80), &SocketConfig{
		DialerBalancerTag: "bal",
	})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	select {
	case outbounds := <-handler.dispatched:
		if len(outbounds) == 0 {
			t.Fatal("expected redirected outbound context")
		}
		if outbounds[len(outbounds)-1].Tag != "proxy-a" {
			t.Fatalf("unexpected redirected tag: %q", outbounds[len(outbounds)-1].Tag)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("expected dispatch through resolved outbound")
	}
}

func TestDialSystemRejectsUnknownDialerBalancerTag(t *testing.T) {
	oldObm := obm
	oldPicker := balancerPicker
	t.Cleanup(func() {
		obm = oldObm
		balancerPicker = oldPicker
	})

	handler := &fakeOutboundHandler{
		tag:        "proxy-a",
		dispatched: make(chan []*session.Outbound, 1),
	}
	obm = &fakeOutboundManager{handler: handler}
	balancerPicker = &fakeBalancerPicker{tag: "bal", resolved: "proxy-a"}

	if _, err := DialSystem(context.Background(), net.TCPDestination(net.DomainAddress("example.com"), 80), &SocketConfig{
		DialerBalancerTag: "missing",
	}); err == nil {
		t.Fatal("expected error for unknown balancer")
	}
}
