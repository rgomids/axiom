package coordination

import (
	"context"
	"errors"
	"time"
)

var ErrRecordNotFound = errors.New("coordination record not found")
var ErrInvalidWait = errors.New("coordination wait must be bounded below caller timeout")

// StreamReader reads canonical streams only; inboxes are deliberately absent.
type StreamReader interface {
	ReadStream(context.Context, string, string) (Stream, bool, error)
}
type Filter struct {
	ParentID, ChildID, AttemptID string
	Kind                         Kind
}
type Reader struct{ Store StreamReader }

func (r Reader) List(ctx context.Context, f Filter) ([]Record, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if r.Store == nil || !validID(f.ParentID) || !validID(f.ChildID) || f.AttemptID != "" && !validID(f.AttemptID) || f.Kind != "" && !validKind(f.Kind) {
		return nil, ErrInvalidRecord
	}
	stream, found, err := r.Store.ReadStream(ctx, f.ParentID, f.ChildID)
	if err != nil {
		return nil, err
	}
	if !found {
		return []Record{}, nil
	}
	if !ValidStream(stream) || stream.ParentID != f.ParentID || stream.ChildID != f.ChildID {
		return nil, ErrInvalidRecord
	}
	records := []Record{}
	for _, record := range stream.Records {
		if (f.AttemptID == "" || f.AttemptID == record.AttemptID) && (f.Kind == "" || f.Kind == record.Kind) {
			wire, err := Encode(record)
			if err != nil {
				return nil, err
			}
			copy, err := Decode(wire)
			if err != nil {
				return nil, err
			}
			records = append(records, copy)
		}
	}
	return records, nil
}
func (r Reader) Show(ctx context.Context, f Filter, id string) (Record, error) {
	if !validID(id) {
		return Record{}, ErrRecordNotFound
	}
	records, err := r.List(ctx, f)
	if err != nil {
		return Record{}, err
	}
	for _, record := range records {
		if record.RecordID == id {
			return record, nil
		}
	}
	return Record{}, ErrRecordNotFound
}

// Wait returns records newer than after. A caller deadline is mandatory.
func (r Reader) Wait(ctx context.Context, f Filter, after uint64, bound time.Duration) ([]Record, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	deadline, ok := ctx.Deadline()
	if !ok || bound <= 0 || bound >= time.Until(deadline) {
		return nil, ErrInvalidWait
	}
	wait, cancel := context.WithTimeout(ctx, bound)
	defer cancel()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		records, err := r.List(wait, f)
		if err != nil {
			return nil, err
		}
		newer := []Record{}
		for _, record := range records {
			if record.Revision > after {
				newer = append(newer, record)
			}
		}
		if len(newer) > 0 {
			return newer, nil
		}
		select {
		case <-wait.Done():
			return nil, wait.Err()
		case <-ticker.C:
		}
	}
}
