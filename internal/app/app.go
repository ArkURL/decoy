// Package app wires the reader and boss screens together.
package app

import (
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/arkurl/cloak/internal/boss"
	"github.com/arkurl/cloak/internal/config"
	"github.com/arkurl/cloak/internal/epub"
	"github.com/arkurl/cloak/internal/reader"
	"github.com/arkurl/cloak/internal/storage"
)

type mode int

const (
	modeReader mode = iota
	modeBoss
)

type idleMsg struct{ token int }

// autoMsg drives the fake session so it keeps updating while hidden.
type autoMsg struct{ token int }

// saveMsg periodically persists progress so it survives a hard exit.
type saveMsg struct{}

// autoInterval is how often the boss screen appends a new line.
const autoInterval = 3 * time.Second

// saveInterval is how often progress is written to disk.
const saveInterval = 10 * time.Second

// idleState is shared by value copies of the model so the timer token survives.
type idleState struct{ token int }

// Model is the root Bubble Tea model.
type Model struct {
	book  *epub.Book
	cfg   config.Config
	keys  config.KeyMap
	store *storage.Store

	reader reader.Model
	boss   boss.Model
	mode   mode

	width, height int
	showHelp      bool
	idle          *idleState
	auto          *idleState
	lastSaved     storage.Position
}

// New builds the root model.
func New(book *epub.Book, cfg config.Config, store *storage.Store) Model {
	keys := config.NewKeyMap(cfg.Keys)
	m := Model{
		book:   book,
		cfg:    cfg,
		keys:   keys,
		store:  store,
		reader: reader.New(book, keys),
		boss:   boss.New(),
		idle:   &idleState{},
		auto:   &idleState{},
	}
	if store != nil {
		if pos, ok := store.Get(book.Path); ok {
			m.reader.Restore(pos)
		}
	}
	return m
}

// Init arms the idle timer and the periodic save.
func (m Model) Init() tea.Cmd {
	return tea.Batch(m.armIdle(), m.armSave())
}

func (m Model) armSave() tea.Cmd {
	return tea.Tick(saveInterval, func(time.Time) tea.Msg { return saveMsg{} })
}

// Position returns the current reading position.
func (m Model) Position() storage.Position { return m.reader.Position() }

func (m Model) armIdle() tea.Cmd {
	if m.idle == nil {
		return nil
	}
	d := time.Duration(m.cfg.IdleSeconds) * time.Second
	if d <= 0 {
		return nil
	}
	m.idle.token++
	tok := m.idle.token
	return tea.Tick(d, func(time.Time) tea.Msg { return idleMsg{tok} })
}

func (m Model) invalidateIdle() { m.idle.token++ }

// armAuto schedules the next fake-session update.
func (m Model) armAuto() tea.Cmd {
	if m.auto == nil {
		return nil
	}
	m.auto.token++
	tok := m.auto.token
	return tea.Tick(autoInterval, func(time.Time) tea.Msg { return autoMsg{tok} })
}

func (m Model) enterBoss() (Model, tea.Cmd) {
	m.mode = modeBoss
	m.boss.SetSize(m.width, m.height)
	m.boss.Blur()
	return m, m.armAuto()
}

func (m Model) exitBoss() Model {
	m.mode = modeReader
	m.auto.token++ // stop the fake-session updates
	return m
}

func (m Model) save() {
	if m.store == nil {
		return
	}
	m.store.Set(m.book.Path, m.book.Title, m.reader.Position())
	_ = m.store.Save()
}

// saveIfChanged persists progress, but only when it actually moved.
func (m *Model) saveIfChanged() {
	if m.store == nil {
		return
	}
	pos := m.reader.Position()
	if pos == m.lastSaved {
		return
	}
	m.store.Set(m.book.Path, m.book.Title, pos)
	if err := m.store.Save(); err == nil {
		m.lastSaved = pos
	}
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.reader.SetSize(m.width, m.height)
		m.boss.SetSize(m.width, m.height)
		return m, nil

	case idleMsg:
		if msg.token == m.idle.token && m.mode == modeReader {
			return m.enterBoss()
		}
		return m, nil

	case autoMsg:
		if msg.token == m.auto.token && m.mode == modeBoss {
			m.boss.Advance()
			return m, m.armAuto()
		}
		return m, nil

	case saveMsg:
		m.saveIfChanged()
		return m, m.armSave()

	case tea.KeyPressMsg:
		return m.handleKey(msg)
	}

	// Forward anything else to whichever screen is active.
	var cmd tea.Cmd
	if m.mode == modeBoss {
		m.boss, cmd = m.boss.Update(msg)
	} else {
		m.reader, cmd = m.reader.Update(msg)
	}
	return m, cmd
}

func (m Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	if m.showHelp {
		m.showHelp = false
		m, cmd := m.rearmIfReader()
		return m, cmd
	}

	var cmds []tea.Cmd

	if m.mode == modeBoss {
		return m.handleBossKey(msg, key)
	}

	// Reader mode.
	switch {
	case key == "ctrl+c" || m.keys.Has(config.Quit, key):
		m.save()
		return m, tea.Quit
	case m.keys.Has(config.Boss, key) || m.keys.Has(config.BossBack, key):
		m.invalidateIdle()
		return m.enterBoss()
	case m.keys.Has(config.Help, key):
		m.showHelp = true
		m, cmd := m.rearmIfReader()
		return m, cmd
	default:
		var cmd tea.Cmd
		m.reader, cmd = m.reader.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	m, idleCmd := m.rearmIfReader()
	if idleCmd != nil {
		cmds = append(cmds, idleCmd)
	}
	return m, tea.Batch(cmds...)
}

func (m Model) handleBossKey(msg tea.KeyPressMsg, key string) (tea.Model, tea.Cmd) {
	// Global escape hatch: works even while the fake input has focus, so the
	// user can always get back to reading (b itself is a normal character
	// while typing).
	if m.keys.Has(config.BossBack, key) {
		m = m.exitBoss()
		m, cmd := m.rearmIfReader()
		return m, cmd
	}

	if m.boss.Focused() {
		switch key {
		case "esc":
			m.boss.Blur()
		case "enter":
			var cmd tea.Cmd
			m.boss, cmd = m.boss.Submit()
			return m, cmd
		case "backspace":
			m.boss.Backspace()
		case "ctrl+c":
			m.save()
			return m, tea.Quit
		default:
			if t := msg.Key().Text; t != "" {
				for _, r := range t {
					m.boss.InsertRune(r)
				}
			}
		}
		return m, nil
	}

	// Browse mode.
	switch {
	case key == "esc" || m.keys.Has(config.Boss, key):
		m = m.exitBoss()
		m, cmd := m.rearmIfReader()
		return m, cmd
	case m.keys.Has(config.BossFocus, key):
		m.boss.Focus()
	case key == "pgup":
		m.boss.ScrollBy(m.boss.PageLines())
	case key == "pgdown":
		m.boss.ScrollBy(-m.boss.PageLines())
	case m.keys.Has(config.Up, key):
		m.boss.ScrollBy(1)
	case m.keys.Has(config.Down, key):
		m.boss.ScrollBy(-1)
	case key == "ctrl+c" || m.keys.Has(config.Quit, key):
		m.save()
		return m, tea.Quit
	}
	return m, nil
}

// rearmIfReader restarts the idle timer when the reading view is visible.
func (m Model) rearmIfReader() (Model, tea.Cmd) {
	if m.mode != modeReader {
		return m, nil
	}
	return m, m.armIdle()
}

// View implements tea.Model.
func (m Model) View() tea.View {
	var content, title string
	switch {
	case m.showHelp:
		content = m.helpView()
		title = "cloak · help"
	case m.mode == modeBoss:
		content = m.boss.View()
		title = m.cfg.FakeTitle
	default:
		content = m.reader.View()
		title = "cloak · " + m.book.Title
	}
	v := tea.NewView(content)
	v.AltScreen = true
	v.WindowTitle = title
	return v
}

func (m Model) helpView() string {
	lines := []string{
		"cloak — 快捷键",
		"",
		"  j / k, ↑ / ↓     上一行 / 下一行",
		"  n / space        下一页        p / pgup        上一页",
		"  ctrl+d / ctrl+u  半页          d / u           半页",
		"  J / ]  K / [     下一章 / 上一章",
		"  g / home  G/end  章首 / 章尾",
		"  t                目录",
		"",
		"  滚动只在章内生效，到末尾按 n 翻到下一章；整页翻页才会自动跨章。",
		"",
		"  b                摸鱼（伪装成 Claude Code）",
		"  ?                本帮助        q / ctrl+c    退出",
		"",
		"摸鱼界面：i 或 enter 输入，enter 发送，esc 返回浏览，b / ctrl+b 切回阅读。",
		"内容为英文并每 3 秒自动追加一行，模拟真实会话。",
		"",
		"按任意键继续…",
	}
	return strings.Join(lines, "\n")
}
