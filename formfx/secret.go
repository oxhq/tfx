package formfx

import (
	"fmt"
	"strings"

	"github.com/oxhq/tfx/internal/share"
	"github.com/oxhq/tfx/runfx"
)

// --- 1. Configuración y Renderer ---

// SecretConfig contiene la configuración declarativa para un SecretPrompt.
type SecretConfig struct {
	Label    string
	Confirm  bool
	Mask     rune // El carácter a usar para enmascarar la entrada.
	Renderer SecretRenderer
}

// DefaultSecretConfig devuelve la configuración por defecto.
func DefaultSecretConfig() SecretConfig {
	return SecretConfig{
		Label:    "Enter secret:",
		Confirm:  false,
		Mask:     '*',
		Renderer: &DefaultSecretRenderer{},
	}
}

// SecretRenderer define la interfaz para renderizar un componente SecretPrompt.
type SecretRenderer interface {
	Render(s *SecretPrompt) []byte
}

// DefaultSecretRenderer es la implementación estándar.
type DefaultSecretRenderer struct{}

func (r *DefaultSecretRenderer) Render(s *SecretPrompt) []byte {
	// Dibuja el label y luego el valor enmascarado del prompt interno.
	maskedValue := strings.Repeat(string(s.Mask), len(s.prompt.Value))
	label := s.Label
	if s.Confirm && s.confirming {
		label = s.Label + " (confirm)"
	}
	return fmt.Appendf(nil, "%s: %s", label, maskedValue)
}

// --- 2. El Componente de Alto Nivel: SecretPrompt ---

// SecretPrompt es un componente de UI para la entrada de texto secreto.
type SecretPrompt struct {
	prompt   *InputPrompt // El primitivo que gestiona el estado del texto.
	Label    string
	Confirm  bool
	Mask     rune
	renderer SecretRenderer

	confirming bool
	firstValue []rune
}

// NewSecretPrompt es el constructor explícito y fuertemente tipado.
func NewSecretPrompt(cfg SecretConfig) (*SecretPrompt, error) {
	p := NewInputPrompt("")

	renderer := cfg.Renderer
	if renderer == nil {
		renderer = &DefaultSecretRenderer{}
	}

	return &SecretPrompt{
		Label:    cfg.Label,
		Confirm:  cfg.Confirm,
		Mask:     cfg.Mask,
		prompt:   p,
		renderer: renderer,
	}, nil
}

// --- 3. La Función de Conveniencia "Multipath" ---

// Secret es la función de conveniencia de alto nivel.
func Secret(opts ...any) (*SecretPrompt, error) {
	cfg, err := share.TryOverloadWithOptions[SecretConfig](opts, DefaultSecretConfig())
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidConfigType, err.Error())
	}
	return NewSecretPrompt(cfg)
}

// --- Métodos del Componente ---

func (s *SecretPrompt) Done() <-chan string {
	return s.prompt.Done
}

func (s *SecretPrompt) Canceled() <-chan struct{} {
	return s.prompt.Canceled
}

// --- Implementación de Interfaces de RunFX ---

func (s *SecretPrompt) Render() []byte {
	return s.renderer.Render(s)
}

func (s *SecretPrompt) OnKey(key runfx.Key) bool {
	if !s.Confirm {
		return s.prompt.OnKey(key)
	}

	switch key.Code {
	case runfx.KeyEscape, runfx.KeyCtrlC:
		close(s.prompt.Canceled)
		return true
	case runfx.KeyEnter:
		if !s.confirming {
			s.firstValue = append(s.firstValue[:0], s.prompt.Value...)
			s.confirming = true
			s.prompt.Value = s.prompt.Value[:0]
			s.prompt.CursorPos = 0
			return false
		}

		if string(s.firstValue) == string(s.prompt.Value) {
			s.prompt.Done <- string(s.prompt.Value)
			return true
		}

		s.confirming = false
		s.firstValue = s.firstValue[:0]
		s.prompt.Value = s.prompt.Value[:0]
		s.prompt.CursorPos = 0
		return false
	}

	return s.prompt.OnKey(key)
}

func (s *SecretPrompt) OnResize(cols, rows int) {}
