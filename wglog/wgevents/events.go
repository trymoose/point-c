package wgevents

import "strings"

//go:generate go run github.com/trymoose/point-c/wglog/wgevents/internal/events-generate

func init() { parser = new(eventParser) }

var _ Parser = (*eventParser)(nil)

type eventParser struct{}

func (*eventParser) ParseUDPGSODisabled(ev *EventUDPGSODisabled, s string, _ ...any) (ok bool) {
	if prefix, suffix, ok := strings.Cut(ev.Format(), "%s"); ok && strings.HasPrefix(s, prefix) && strings.HasSuffix(s, suffix) {
		ev.OnLAddr = strings.TrimPrefix(strings.TrimSuffix(s, suffix), prefix)
		return true
	}
	return false
}
