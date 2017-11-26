package telebot

import (
	"time"

	"github.com/pkg/errors"
)

// Poller is a provider of Updates.
//
// All pollers must implement Poll(), which accepts bot
// pointer and subscription channel and start polling
// synchronously straight away.
type Poller interface {
	// Poll is supposed to take the bot object
	// subscription channel and start polling
	// for Updates immediately.
	Poll(b *Bot, updates chan Update, stop chan struct{})
}

// MiddlewarePoller is a special kind of poller that acts
// like a filter for updates. It could be used for spam
// handling, banning or whatever.
//
// For heavy middleware, use increased capacity.
//
type MiddlewarePoller struct {
	Poller Poller
	filter func(*Update) bool

	Capacity int // Default: 1
}

// Middleware constructs a middleware poller.
func Middleware(p Poller, filter func(*Update) bool) *MiddlewarePoller {
	return &MiddlewarePoller{
		Poller: p,
		filter: filter,
	}
}

// Poll sieves updates through middleware filter.
func (p *MiddlewarePoller) Poll(b *Bot, dest chan Update, stop chan struct{}) {
	cap := 1
	if p.Capacity > 1 {
		cap = p.Capacity
	}

	middle := make(chan Update, cap)
	stop2 := make(chan struct{})

	go p.Poller.Poll(b, middle, stop2)

	for {
		select {
		case <-stop:
			close(stop2)
			return
		case upd := <-middle:
			if p.filter(&upd) {
				dest <- upd
			}
		}
	}
}

// LongPoller is a classic LongPoller with timeout.
type LongPoller struct {
	Timeout time.Duration
}

// Poll does long polling.
func (p *LongPoller) Poll(b *Bot, dest chan Update, stop chan struct{}) {
	var latestUpd int

	for {
		select {
		case <-stop:
			return
		default:
			updates, err := b.getUpdates(latestUpd+1, p.Timeout)

			if err != nil {
				b.debug(errors.Wrap(err, "getUpdates() failed"))
				continue
			}

			for _, update := range updates {
				latestUpd = update.ID
				dest <- update
			}
		}
	}
}