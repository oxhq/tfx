package logfx

import (
	"maps"

	"github.com/oxhq/tfx/internal/share"
)

func cloneShareFields(fields map[string]any) share.Fields {
	clone := make(share.Fields, len(fields))
	maps.Copy(clone, fields)
	return clone
}

func cloneAnyFields(fields map[string]any) map[string]any {
	clone := make(map[string]any, len(fields))
	for key, value := range fields {
		clone[key] = value
	}
	return clone
}

func mergeShareFields(base share.Fields, overlays ...share.Fields) share.Fields {
	merged := make(share.Fields, len(base))
	maps.Copy(merged, base)
	for _, overlay := range overlays {
		maps.Copy(merged, overlay)
	}
	return merged
}

func (l *Logger) snapshotWriters() ([]share.Writer, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	writers := append([]share.Writer(nil), l.writers...)
	return writers, l.options.Async
}

func (l *Logger) dispatch(entry *share.Entry) {
	writers, async := l.snapshotWriters()
	if async {
		for _, writer := range writers {
			l.wg.Add(1)
			go func(w share.Writer, e *share.Entry) {
				defer l.wg.Done()
				_ = w.Write(e)
			}(writer, entry)
		}
		return
	}

	for _, writer := range writers {
		_ = writer.Write(entry)
	}
}
