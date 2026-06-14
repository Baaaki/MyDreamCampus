package eventbus

import (
	"fmt"

	"github.com/baaaki/mydreamcampus/monolith/internal/platform/rabbitmq"
)

// ModuleExchanges lists every per-module topic exchange the monolith owns.
// New modules must add their exchange here so the topology declare loop
// in main.go covers them; the value is also the publisher target string.
var ModuleExchanges = []string{
	"auth.events",
	"staff.events",
	"student.events",
	"course_catalog.events",
	"enrollment.events",
	"attendance.events",
	"grades.events",
	"meal.events",
	"payment.events",
}

// DeclareModuleExchanges declares all module exchanges as durable topics.
// Idempotent — RabbitMQ's ExchangeDeclare is a no-op when the exchange
// already exists with the same arguments.
func DeclareModuleExchanges(publisher *rabbitmq.Publisher) error {
	for _, exchange := range ModuleExchanges {
		if err := publisher.DeclareExchange(exchange); err != nil {
			return fmt.Errorf("declare exchange %s: %w", exchange, err)
		}
	}
	return nil
}

// DownstreamBinding describes a (queue, exchange, routing_key) triple that
// the monolith pre-declares so messages aren't lost while the consumer is
// offline (plan section 5.6.3).
type DownstreamBinding struct {
	Queue      string
	Exchange   string
	RoutingKey string
}

// DeclareDownstreamBindings creates each consumer queue and binds it to the
// matching module exchange. Called from main.go on startup.
func DeclareDownstreamBindings(publisher *rabbitmq.Publisher, bindings []DownstreamBinding) error {
	for _, b := range bindings {
		if err := publisher.DeclareAndBindQueue(b.Queue, b.Exchange, b.RoutingKey); err != nil {
			return fmt.Errorf("bind %s -> %s/%s: %w", b.Queue, b.Exchange, b.RoutingKey, err)
		}
	}
	return nil
}
