package runfx

import (
	"sync"
	"sync/atomic"
)

// VisualID is a unique and opaque identifier for a mounted visual component.
type VisualID uint64

// Multiplexer safely manages a set of visual components.
// Now uses an atomic counter to generate unique IDs.
type Multiplexer struct {
	nextID  uint64
	order   []VisualID
	visuals map[VisualID]Visual
	mu      sync.Mutex
}

// NewMultiplexer creates a new instance of the multiplexer.
func NewMultiplexer() *Multiplexer {
	return &Multiplexer{
		order:   make([]VisualID, 0),
		visuals: make(map[VisualID]Visual),
	}
}

// Mount registers a new visual component and assigns it a unique ID.
// Returns the assigned ID.
func (m *Multiplexer) Mount(v Visual) VisualID {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Atomically generates a unique ID.
	id := VisualID(atomic.AddUint64(&m.nextID, 1))
	m.visuals[id] = v
	m.order = append(m.order, id)
	return id
}

// Unmount removes a visual component using its ID.
func (m *Multiplexer) Unmount(id VisualID) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.visuals, id)
	for i, existing := range m.order {
		if existing == id {
			m.order = append(m.order[:i], m.order[i+1:]...)
			break
		}
	}
}

// GetVisual retrieves a visual component by its ID.
func (m *Multiplexer) GetVisual(id VisualID) (Visual, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	v, ok := m.visuals[id]
	return v, ok
}

// ListVisuals returns a list of the IDs of all mounted components.
func (m *Multiplexer) ListVisuals() []VisualID {
	m.mu.Lock()
	defer m.mu.Unlock()
	ids := make([]VisualID, len(m.order))
	copy(ids, m.order)
	return ids
}

// Render adds the bytes of all visual components for a single frame.
func (m *Multiplexer) Render() []byte {
	m.mu.Lock()
	defer m.mu.Unlock()

	var out []byte
	for _, id := range m.order {
		if v := m.visuals[id]; v != nil {
			out = append(out, v.Render()...)
		}
	}
	return out
}

// OnResize notifies all visual components of a terminal resize event.
func (m *Multiplexer) OnResize(cols, rows int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, id := range m.order {
		if v := m.visuals[id]; v != nil {
			v.OnResize(cols, rows)
		}
	}
}
