package db

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"challenge/kit/broker"
)

type Record struct {
	AggregateID string
	EventName   string
	Payload     []byte
	OccurredAt  time.Time
}

type Store struct {
	mu      sync.RWMutex
	streams map[string][]Record
	log     []Record
	fileMu  sync.Mutex
	f       *os.File
}

func New() *Store {
	return &Store{streams: make(map[string][]Record)}
}

func NewWithFile(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		log.Printf("layer=store component=db method=NewWithFile path=%s err=%v", path, err)
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o644)
	if err != nil {
		log.Printf("layer=store component=db method=NewWithFile path=%s err=%v", path, err)
		return nil, err
	}

	s := &Store{streams: make(map[string][]Record), f: f}
	if err := s.replayFromFile(path, f); err != nil {
		_ = f.Close()
		return nil, err
	}
	if _, err := f.Seek(0, io.SeekEnd); err != nil {
		log.Printf("layer=store component=db method=NewWithFile path=%s err=%v", path, err)
		_ = f.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) replayFromFile(path string, f *os.File) error {
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		log.Printf("layer=store component=db method=replayFromFile path=%s err=%v", path, err)
		return err
	}

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var raw struct {
			AggregateID string          `json:"aggregate_id"`
			EventName   string          `json:"event_name"`
			Payload     json.RawMessage `json:"payload"`
			OccurredAt  time.Time       `json:"occurred_at"`
		}
		if err := json.Unmarshal(line, &raw); err != nil {
			log.Printf("layer=store component=db method=replayFromFile path=%s err=%v", path, err)
			return err
		}

		rec := Record{
			AggregateID: raw.AggregateID,
			EventName:   raw.EventName,
			Payload:     []byte(raw.Payload),
			OccurredAt:  raw.OccurredAt,
		}
		s.mu.Lock()
		s.streams[raw.AggregateID] = append(s.streams[raw.AggregateID], rec)
		s.log = append(s.log, rec)
		s.mu.Unlock()
	}
	if err := scanner.Err(); err != nil {
		log.Printf("layer=store component=db method=replayFromFile path=%s err=%v", path, err)
		return err
	}
	return nil
}

func (s *Store) Close() error {
	s.fileMu.Lock()
	defer s.fileMu.Unlock()
	if s.f == nil {
		return nil
	}
	err := s.f.Close()
	if err != nil {
		log.Printf("layer=store component=db method=Close err=%v", err)
	}
	s.f = nil
	return err
}

func (s *Store) Append(ctx context.Context, aggregateID string, evt broker.Event) error {
	payload, err := json.Marshal(evt)
	if err != nil {
		log.Printf("layer=store component=db method=Append aggregate_id=%s event=%s err=%v", aggregateID, evt.Name(), err)
		return err
	}

	occurredAt := time.Now().UTC()

	s.mu.Lock()
	rec := Record{
		AggregateID: aggregateID,
		EventName:   evt.Name(),
		Payload:     payload,
		OccurredAt:  occurredAt,
	}
	s.streams[aggregateID] = append(s.streams[aggregateID], rec)
	s.log = append(s.log, rec)
	s.mu.Unlock()

	s.fileMu.Lock()
	if s.f != nil {
		line := map[string]any{
			"aggregate_id": aggregateID,
			"event_name":   evt.Name(),
			"payload":      json.RawMessage(payload),
			"occurred_at":  occurredAt,
		}
		b, mErr := json.Marshal(line)
		if mErr != nil {
			log.Printf("layer=store component=db method=Append aggregate_id=%s event=%s err=%v", aggregateID, evt.Name(), mErr)
		} else {
			if _, wErr := s.f.Write(append(b, '\n')); wErr != nil {
				log.Printf("layer=store component=db method=Append aggregate_id=%s event=%s err=%v", aggregateID, evt.Name(), wErr)
			}
		}
	}
	s.fileMu.Unlock()
	return nil
}

func (s *Store) Load(ctx context.Context, aggregateID string) []Record {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]Record(nil), s.streams[aggregateID]...)
}

func (s *Store) All(ctx context.Context) []Record {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]Record(nil), s.log...)
}
