// Package shared contains shared value objects and types across domains
package shared

import (
	"errors"
	"time"
)

// Core domain interfaces
type ValueObject interface {
	Equals(other ValueObject) bool
	Validate() error
}

type AggregateRoot interface {
	GetID() string
	GetEvents() []DomainEvent
	ClearEvents()
	ApplyEvent(event DomainEvent)
}

type DomainEvent interface {
	GetEventType() string
	GetTimestamp() time.Time
	GetAggregateID() string
}

type Repository[T AggregateRoot] interface {
	Save(aggregate T) error
	GetByID(id string) (T, error)
	Delete(id string) error
}

// UserID represents a user identifier
type UserID string

func (u UserID) String() string {
	return string(u)
}

func (u UserID) IsEmpty() bool {
	return string(u) == ""
}

// TenantID represents a tenant identifier
type TenantID string

func (t TenantID) String() string {
	return string(t)
}

func (t TenantID) IsEmpty() bool {
	return string(t) == ""
}

// TeamID represents a team identifier
type TeamID string

func (t TeamID) String() string {
	return string(t)
}

func (t TeamID) IsEmpty() bool {
	return string(t) == ""
}

// SystemUserID represents system operations
const SystemUserID UserID = "system"

// DateRange represents a date range value object
type DateRange struct {
	Start time.Time
	End   time.Time
}

func NewDateRange(start, end time.Time) (*DateRange, error) {
	if start.After(end) {
		return nil, errors.New("start date cannot be after end date")
	}
	return &DateRange{Start: start, End: end}, nil
}

func (dr *DateRange) Contains(t time.Time) bool {
	return !t.Before(dr.Start) && !t.After(dr.End)
}

func (dr *DateRange) Duration() time.Duration {
	return dr.End.Sub(dr.Start)
}

// EventBus interface for publishing domain events
type EventBus interface {
	Publish(event interface{}) error
	Subscribe(eventType string, handler EventHandler) error
}

// EventHandler handles domain events
type EventHandler interface {
	Handle(event interface{}) error
}

// Money value object for monetary amounts
type Money struct {
	Amount   int64  // Amount in smallest currency unit (e.g., cents)
	Currency string // Currency code (e.g., "USD", "CNY")
}

func NewMoney(amount int64, currency string) (*Money, error) {
	if currency == "" {
		return nil, errors.New("currency cannot be empty")
	}
	return &Money{Amount: amount, Currency: currency}, nil
}

func (m *Money) IsZero() bool {
	return m.Amount == 0
}

func (m *Money) Add(other *Money) (*Money, error) {
	if m.Currency != other.Currency {
		return nil, errors.New("cannot add money with different currencies")
	}
	return &Money{Amount: m.Amount + other.Amount, Currency: m.Currency}, nil
}

// Email value object
type Email struct {
	value string
}

func NewEmail(email string) (*Email, error) {
	if email == "" {
		return nil, errors.New("email cannot be empty")
	}
	// Add email validation logic here
	return &Email{value: email}, nil
}

func (e *Email) Value() string {
	return e.value
}

// PhoneNumber value object
type PhoneNumber struct {
	value string
}

func NewPhoneNumber(phone string) (*PhoneNumber, error) {
	if phone == "" {
		return nil, errors.New("phone number cannot be empty")
	}
	// Add phone validation logic here
	return &PhoneNumber{value: phone}, nil
}

func (p *PhoneNumber) Value() string {
	return p.value
}