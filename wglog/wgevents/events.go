package wgevents

//go:generate go run github.com/trymoose/point-c/wglog/wgevents/internal/events-generate

func init() { parser = new(eventParser) }

var _ Parser = (*eventParser)(nil)

type eventParser struct{}

func (*eventParser) ParseUDPGSODisabled(ev *EventUDPGSODisabled, s string, _ ...any) bool {

}
