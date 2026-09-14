// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwobservability

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/signalfx/signalfx-go/directory"
	"github.com/signalfx/signalfx-go/template"
)

// templateAPIStore is a minimal in-memory fake of the Template API used to
// exercise the full Create/Read/Update/Delete lifecycle of the
// observability_dashboard and observability_template resources against a
// mock HTTP server.
type templateAPIStore struct {
	mu    sync.Mutex
	next  int
	items map[string]*template.Template
}

func newTemplateAPIStore() *templateAPIStore {
	return &templateAPIStore{items: make(map[string]*template.Template)}
}

func (s *templateAPIStore) handlers() map[string]http.Handler {
	return map[string]http.Handler{
		"POST /v2/template":        http.HandlerFunc(s.create),
		"GET /v2/template/{id}":    http.HandlerFunc(s.read),
		"PUT /v2/template/{id}":    http.HandlerFunc(s.update),
		"DELETE /v2/template/{id}": http.HandlerFunc(s.delete),
	}
}

func (s *templateAPIStore) create(w http.ResponseWriter, r *http.Request) {
	var write template.Write
	if err := json.NewDecoder(r.Body).Decode(&write); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	s.next++
	record := &template.Template{
		ID:       fmt.Sprintf("template-%d", s.next),
		Type:     write.Type,
		Title:    write.Title,
		Spec:     write.Spec,
		Metadata: &template.Metadata{RootElement: write.Metadata.RootElement, Imports: write.Metadata.Imports},
	}
	s.items[record.ID] = record
	s.mu.Unlock()

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(template.Result{Data: record})
}

func (s *templateAPIStore) read(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	record, ok := s.items[r.PathValue("id")]
	s.mu.Unlock()
	if !ok {
		http.Error(w, "template not found", http.StatusNotFound)
		return
	}
	_ = json.NewEncoder(w).Encode(template.Result{Data: record})
}

func (s *templateAPIStore) update(w http.ResponseWriter, r *http.Request) {
	var write template.Write
	if err := json.NewDecoder(r.Body).Decode(&write); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	record, ok := s.items[r.PathValue("id")]
	if ok {
		record.Type = write.Type
		record.Title = write.Title
		record.Spec = write.Spec
		record.Metadata = &template.Metadata{RootElement: write.Metadata.RootElement, Imports: write.Metadata.Imports}
	}
	s.mu.Unlock()
	if !ok {
		http.Error(w, "template not found", http.StatusNotFound)
		return
	}

	_ = json.NewEncoder(w).Encode(template.Result{Data: record})
}

func (s *templateAPIStore) delete(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	delete(s.items, r.PathValue("id"))
	s.mu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

// directoryAPIStore is a minimal in-memory fake of the Directory API used to
// exercise the full Create/Read/Update/Delete lifecycle of the
// observability_directory resource against a mock HTTP server.
type directoryAPIStore struct {
	mu      sync.Mutex
	entries map[string]*directory.Entry
}

func newDirectoryAPIStore() *directoryAPIStore {
	return &directoryAPIStore{entries: make(map[string]*directory.Entry)}
}

func (s *directoryAPIStore) handlers() map[string]http.Handler {
	return map[string]http.Handler{
		"GET /v2/directory/{path...}":    http.HandlerFunc(s.read),
		"PATCH /v2/directory/{path...}":  http.HandlerFunc(s.patch),
		"DELETE /v2/directory/{path...}": http.HandlerFunc(s.delete),
	}
}

func (s *directoryAPIStore) read(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	entry, ok := s.entries[r.PathValue("path")]
	s.mu.Unlock()
	if !ok {
		http.Error(w, "directory entry not found", http.StatusNotFound)
		return
	}
	_ = json.NewEncoder(w).Encode(directory.Result{Data: entry})
}

func (s *directoryAPIStore) patch(w http.ResponseWriter, r *http.Request) {
	var patch directory.Patch
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	path := r.PathValue("path")
	s.mu.Lock()
	entry, ok := s.entries[path]
	if !ok {
		entry = &directory.Entry{Path: path}
		s.entries[path] = entry
	}
	if patch.Pinned != nil {
		entry.Pinned = *patch.Pinned
	}
	if patch.Templates != nil {
		entry.Templates = *patch.Templates
	}
	s.mu.Unlock()

	_ = json.NewEncoder(w).Encode(directory.Result{Data: entry})
}

func (s *directoryAPIStore) delete(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	delete(s.entries, r.PathValue("path"))
	s.mu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}
