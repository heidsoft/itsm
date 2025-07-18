// Code generated by ent, DO NOT EDIT.

package ent

import (
	"context"
	"itsm-backend/ent/flowinstance"
	"itsm-backend/ent/predicate"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
)

// FlowInstanceDelete is the builder for deleting a FlowInstance entity.
type FlowInstanceDelete struct {
	config
	hooks    []Hook
	mutation *FlowInstanceMutation
}

// Where appends a list predicates to the FlowInstanceDelete builder.
func (fid *FlowInstanceDelete) Where(ps ...predicate.FlowInstance) *FlowInstanceDelete {
	fid.mutation.Where(ps...)
	return fid
}

// Exec executes the deletion query and returns how many vertices were deleted.
func (fid *FlowInstanceDelete) Exec(ctx context.Context) (int, error) {
	return withHooks(ctx, fid.sqlExec, fid.mutation, fid.hooks)
}

// ExecX is like Exec, but panics if an error occurs.
func (fid *FlowInstanceDelete) ExecX(ctx context.Context) int {
	n, err := fid.Exec(ctx)
	if err != nil {
		panic(err)
	}
	return n
}

func (fid *FlowInstanceDelete) sqlExec(ctx context.Context) (int, error) {
	_spec := sqlgraph.NewDeleteSpec(flowinstance.Table, sqlgraph.NewFieldSpec(flowinstance.FieldID, field.TypeInt))
	if ps := fid.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	affected, err := sqlgraph.DeleteNodes(ctx, fid.driver, _spec)
	if err != nil && sqlgraph.IsConstraintError(err) {
		err = &ConstraintError{msg: err.Error(), wrap: err}
	}
	fid.mutation.done = true
	return affected, err
}

// FlowInstanceDeleteOne is the builder for deleting a single FlowInstance entity.
type FlowInstanceDeleteOne struct {
	fid *FlowInstanceDelete
}

// Where appends a list predicates to the FlowInstanceDelete builder.
func (fido *FlowInstanceDeleteOne) Where(ps ...predicate.FlowInstance) *FlowInstanceDeleteOne {
	fido.fid.mutation.Where(ps...)
	return fido
}

// Exec executes the deletion query.
func (fido *FlowInstanceDeleteOne) Exec(ctx context.Context) error {
	n, err := fido.fid.Exec(ctx)
	switch {
	case err != nil:
		return err
	case n == 0:
		return &NotFoundError{flowinstance.Label}
	default:
		return nil
	}
}

// ExecX is like Exec, but panics if an error occurs.
func (fido *FlowInstanceDeleteOne) ExecX(ctx context.Context) {
	if err := fido.Exec(ctx); err != nil {
		panic(err)
	}
}
