package formfx

import "github.com/oxhq/tfx/runfx"

// --- 1. El Componente Primitivo: InputPrompt ---

// InputPrompt es un componente primitivo que gestiona el estado de una entrada de texto.
// Es agnóstico a su representación visual (texto plano, contraseña, etc.).
type InputPrompt struct {
	Value      []rune // Usamos runas para un manejo correcto de caracteres multibyte.
	CursorPos  int    // Posición del cursor dentro del slice de runas.
	keyHandler TextKeyHandlerFunc

	Done     chan string // Devuelve el valor final como un string.
	Canceled chan struct{}
}

type TextKeyHandlerFunc func(p *InputPrompt, key runfx.Key) bool

// NewInputPrompt crea una nueva instancia de un InputPrompt primitivo.
func NewInputPrompt(defaultValue string) *InputPrompt {
	return &InputPrompt{
		Value:      []rune(defaultValue),
		CursorPos:  len(defaultValue),
		Done:       make(chan string, 1),
		Canceled:   make(chan struct{}),
		keyHandler: TextInputKeyHandler, // Usa un manejador de teclado para texto por defecto.
	}
}

// SetKeyHandler permite inyectar una lógica de teclado personalizada.
func (p *InputPrompt) SetKeyHandler(h TextKeyHandlerFunc) {
	p.keyHandler = h
}

// --- 2. Lógica de Teclado por Defecto ---

// TextInputKeyHandler proporciona un manejo de teclado básico para la edición de texto.
func TextInputKeyHandler(p *InputPrompt, key runfx.Key) bool {
	switch key.Code {
	case runfx.KeyEnter:
		p.Done <- string(p.Value)
		return true // Detener el loop.
	case runfx.KeyEscape, runfx.KeyCtrlC:
		close(p.Canceled)
		return true // Detener el loop.
	case runfx.KeyBackspace:
		if p.CursorPos > 0 {
			p.Value = append(p.Value[:p.CursorPos-1], p.Value[p.CursorPos:]...)
			p.CursorPos--
		}
	case runfx.KeyArrowLeft:
		if p.CursorPos > 0 {
			p.CursorPos--
		}
	case runfx.KeyArrowRight:
		if p.CursorPos < len(p.Value) {
			p.CursorPos++
		}
	default:
		inserted := key.Rune
		if inserted == 0 && key.Code == runfx.KeySpace {
			inserted = ' '
		}
		if inserted != 0 {
			p.Value = append(
				p.Value[:p.CursorPos],
				append([]rune{inserted}, p.Value[p.CursorPos:]...)...)
			p.CursorPos++
		} else if !key.IsPrintable() {
			return false
		}
	}
	return false // El loop debe continuar.
}

// --- 3. Conformidad con la Interfaz de RunFX ---

func (p *InputPrompt) OnKey(key runfx.Key) bool {
	if p.keyHandler != nil {
		return p.keyHandler(p, key)
	}
	return TextInputKeyHandler(p, key)
}

func (p *InputPrompt) Render() []byte          { return nil }
func (p *InputPrompt) OnResize(cols, rows int) {}
