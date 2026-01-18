package ui

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/charmbracelet/harmonica"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SpringAnimation gerencia animações spring usando Harmonica
type SpringAnimation struct {
	spring   harmonica.Spring
	value    float64
	velocity float64
	target   float64
}

// NewSpringAnimation cria uma nova animação spring
// fps: frames por segundo (30 ou 60)
// stiffness: rigidez da mola (angularFrequency) - valores típicos: 5.0-10.0
// damping: amortecimento (dampingRatio) - valores típicos: 0.5-0.8
func NewSpringAnimation(fps int, stiffness, damping float64) SpringAnimation {
	deltaTime := harmonica.FPS(fps)
	return SpringAnimation{
		spring:   harmonica.NewSpring(deltaTime, stiffness, damping),
		value:    0.0,
		velocity: 0.0,
		target:   0.0,
	}
}

// Update atualiza a animação
// dt é em segundos (float64) para compatibilidade com Harmonica
func (s *SpringAnimation) Update(dt float64) {
	s.value, s.velocity = s.spring.Update(s.value, s.velocity, s.target)
}

// SetTarget define o valor alvo da animação
func (s *SpringAnimation) SetTarget(target float64) {
	s.target = target
}

// Value retorna o valor atual da animação
func (s *SpringAnimation) Value() float64 {
	return s.value
}

// IsAnimating verifica se a animação ainda está em movimento
func (s *SpringAnimation) IsAnimating() bool {
	// Considerar animando se ainda não chegou ao alvo (tolerância de 0.01)
	return math.Abs(s.value-s.target) > 0.01 || math.Abs(s.velocity) > 0.01
}

// HeaderAnimationModel gerencia animação de entrada para headers
type HeaderAnimationModel struct {
	animation SpringAnimation
	opacity   float64
	width     int
	done      bool
}

// HeaderAnimationMsg é uma mensagem para atualizar a animação do header
// Delta é em segundos (float64) para compatibilidade com Harmonica
type HeaderAnimationMsg struct {
	Delta float64
}

// NewHeaderAnimationModel cria um novo modelo de animação de header
func NewHeaderAnimationModel() HeaderAnimationModel {
	anim := NewSpringAnimation(60, 6.0, 0.7)
	anim.SetTarget(1.0)

	return HeaderAnimationModel{
		animation: anim,
		opacity:   0.0,
		width:     60,
		done:      false,
	}
}

func (m HeaderAnimationModel) Init() tea.Cmd {
	return tea.Tick(16*time.Millisecond, func(t time.Time) tea.Msg {
		// Converter duração para float64 (segundos)
		dtSeconds := float64(16 * time.Millisecond) / float64(time.Second)
		return HeaderAnimationMsg{Delta: dtSeconds}
	})
}

func (m HeaderAnimationModel) Update(msg tea.Msg) (HeaderAnimationModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, nil

	case HeaderAnimationMsg:
		m.animation.Update(msg.Delta)
		m.opacity = m.animation.Value()

		// Se muito próximo do alvo, considerar completo
		if !m.animation.IsAnimating() && m.opacity >= 0.99 {
			m.done = true
			return m, nil
		}

		return m, tea.Tick(16*time.Millisecond, func(t time.Time) tea.Msg {
			dtSeconds := float64(16 * time.Millisecond) / float64(time.Second)
			return HeaderAnimationMsg{Delta: dtSeconds}
		})

	default:
		return m, nil
	}
}

func (m HeaderAnimationModel) View(title string) string {
	if m.opacity < 0.01 {
		return ""
	}

	// Renderizar header diretamente (sem opacity, que não existe no lipgloss)
	// Em vez disso, podemos variar a intensidade pela cor ou apenas renderizar quando completo
	if m.opacity < 0.5 {
		// Usar cor mais fraca durante animação
		style := HeaderStyle.Copy().Foreground(MutedColor)
		return style.Render(title)
	}

	return HeaderStyle.Render(title)
}

// ProgressAnimationModel gerencia animação suave para barras de progresso
type ProgressAnimationModel struct {
	animation SpringAnimation
	current   float64
	target    float64
	width     int
}

// NewProgressAnimationModel cria um novo modelo de animação de progresso
func NewProgressAnimationModel() ProgressAnimationModel {
	anim := NewSpringAnimation(60, 8.0, 0.6)

	return ProgressAnimationModel{
		animation: anim,
		current:   0.0,
		target:    0.0,
		width:     60,
	}
}

// SetProgress define o progresso alvo (0.0 a 1.0)
func (m *ProgressAnimationModel) SetProgress(progress float64) {
	if progress < 0.0 {
		progress = 0.0
	}
	if progress > 1.0 {
		progress = 1.0
	}
	m.target = progress
	m.animation.SetTarget(progress)
}

// Update atualiza a animação
// dt é em segundos (float64) para compatibilidade com Harmonica
func (m *ProgressAnimationModel) Update(dt float64) {
	m.animation.Update(dt)
	m.current = m.animation.Value()
}

// View renderiza a barra de progresso animada
func (m ProgressAnimationModel) View() string {
	percent := int(m.current * 100)
	if percent > 100 {
		percent = 100
	}
	if percent < 0 {
		percent = 0
	}

	// Garantir que current está entre 0.0 e 1.0
	current := m.current
	if current < 0.0 {
		current = 0.0
	}
	if current > 1.0 {
		current = 1.0
	}

	// Garantir que width é positivo
	width := m.width
	if width < 0 {
		width = 0
	}

	filled := int(float64(width) * current)
	empty := width - filled

	// Garantir que filled e empty não são negativos
	if filled < 0 {
		filled = 0
	}
	if empty < 0 {
		empty = 0
	}

	// Cor baseada no progresso
	var barColor lipgloss.Color
	if current < 0.3 {
		barColor = WarningColor
	} else if current < 0.7 {
		barColor = InfoColor
	} else {
		barColor = SuccessColor
	}

	filledBar := lipgloss.NewStyle().
		Foreground(barColor).
		Render(strings.Repeat("█", filled))

	emptyBar := lipgloss.NewStyle().
		Foreground(MutedColorDark).
		Render(strings.Repeat("░", empty))

	percentStr := fmt.Sprintf("%3d%%", percent)
	percentStyle := lipgloss.NewStyle().
		Foreground(TextSecondary).
		Bold(true).
		PaddingLeft(1)

	return fmt.Sprintf("[%s%s] %s", filledBar, emptyBar, percentStyle.Render(percentStr))
}

// PulsingText cria um texto com efeito pulsante
// cycle deve estar entre 0.0 e 1.0 para um ciclo completo
func PulsingText(text string, cycle float64) string {
	// Usar ciclo para alternar entre cores (sem opacity que não existe)
	// Alternar entre PrimaryColor e AccentColor baseado no ciclo
	var color lipgloss.Color
	// Usar seno do ciclo para transição suave entre cores
	sinValue := math.Sin(cycle * 2 * math.Pi)
	if sinValue > 0 {
		color = PrimaryColor
	} else {
		color = AccentColor
	}

	style := lipgloss.NewStyle().
		Foreground(color).
		Bold(true)

	return style.Render(text)
}
