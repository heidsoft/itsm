// Package domain provides the core domain logic for ITSM system
// Following DDD (Domain Driven Design) principles
package domain

// Domain Event represents an event that occurred within the domain
type DomainEvent interface {
	GetEventType() string
	GetAggregateID() string
	GetEventID() string
	GetTimestamp() int64
	GetEventData() interface{}
}

// AggregateRoot is the base interface for all aggregate roots
type AggregateRoot interface {
	GetID() string
	GetVersion() int
	GetEvents() []DomainEvent
	ClearEvents()
	ApplyEvent(event DomainEvent)
}

// Repository defines the contract for data persistence
type Repository[T AggregateRoot] interface {
	Save(aggregate T) error
	GetByID(id string) (T, error)
	Delete(id string) error
}

// DomainService defines business logic that doesn't belong to any specific aggregate
type DomainService interface {
	GetServiceName() string
}

// ValueObject represents a domain value object
type ValueObject interface {
	Equals(other ValueObject) bool
	Validate() error
}

// Entity represents a domain entity with identity
type Entity interface {
	GetID() string
	Equals(other Entity) bool
}