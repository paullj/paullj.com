package tui

import (
	"fmt"
	"image/color"
	"sort"
	"strings"
	"time"

	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/paullj/paullj.com/internal/config"
	"github.com/paullj/paullj.com/internal/content"
	"github.com/paullj/paullj.com/internal/images"
)

type tab int

const (
	homeTab tab = iota
	aboutTab
	postsTab
	numTabs = 3
)

var tabNames = [numTabs]string{"Home", "About", "Posts"}

type homeSection int

const (
	homeSectionLinks homeSection = iota
	homeSectionPosts
)

type theme int

const (
	themeDark theme = iota
	themeLight
)

func (t theme) String() string {
	if t == themeLight {
		return "light"
	}
	return "dark"
}

type sortOrder int

const (
	sortNewest sortOrder = iota
	sortOldest
)

type Model struct {
	posts    []content.Post
	cfg      *config.Config
	aboutRaw string

	activeTab   tab
	homeSection homeSection
	homeLinkIdx int
	homePostIdx int

	inPostDetail bool
	returnTab    tab

	// Posts tab state
	postsIdx       int
	postsOffset    int
	postsFilter    textinput.Model
	postsFiltering bool
	postsSort      sortOrder

	aboutVP  viewport.Model
	aboutOK  bool
	postVP   viewport.Model
	currPost *content.Post

	width  int
	height int
	ready  bool
	theme  theme

	imageMode images.ImageMode
	cache     *images.Cache

	splash   splashState
	status   string
	statusID int
}

func NewModel(
	posts []content.Post,
	imageMode images.ImageMode,
	cfg *config.Config,
	aboutRaw string,
	cache *images.Cache,
) Model {
	ti := textinput.New()
	ti.Prompt = "/ "
	ti.CharLimit = 64

	initSection := homeSectionLinks
	if len(cfg.Content.Links) == 0 {
		initSection = homeSectionPosts
	}

	return Model{
		posts:       posts,
		cfg:         cfg,
		aboutRaw:    aboutRaw,
		imageMode:   imageMode,
		cache:       cache,
		postsFilter: ti,
		homeSection: initSection,
	}
}

type clearStatusMsg struct{ id int }

func (m Model) setStatus(msg string) (Model, tea.Cmd) {
	m.statusID++
	m.status = msg
	id := m.statusID
	return m, tea.Tick(2*time.Second, func(time.Time) tea.Msg {
		return clearStatusMsg{id: id}
	})
}

func (m Model) Init() tea.Cmd {
	if m.cfg.SSH.Splash.Text == "" {
		m.splash.done = true
		return nil
	}
	return m.splashTick()
}

func (m Model) viewportHeight() int {
	tabBarH := lipgloss.Height(m.renderTabBar(m.contentWidth()))
	footerH := 2 // separator + help text
	h := m.height - tabBarH - footerH
	if h < 1 {
		h = 1
	}
	return h
}

func (m Model) contentWidth() int {
	w := m.cfg.SSH.MaxWidth
	if m.width < w {
		w = m.width
	}
	return w
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		cw := m.contentWidth()
		vpH := m.viewportHeight()
		if m.inPostDetail {
			m.postVP.SetWidth(cw)
			m.postVP.SetHeight(vpH)
		}
		if m.aboutOK {
			m.aboutVP.SetWidth(cw)
			m.aboutVP.SetHeight(vpH)
		}
		m.ready = true
		return m, nil
	case clearStatusMsg:
		if msg.id == m.statusID {
			m.status = ""
		}
		return m, nil
	case splashTickMsg, splashDoneMsg:
		if !m.splash.done {
			return m.updateSplash(msg)
		}
		return m, nil

	case tea.KeyPressMsg:
		if !m.splash.done {
			if m.cfg.SSH.Splash.SkipOnKey {
				m.splash.done = true
				m.splash.charIndex = len(m.cfg.SSH.Splash.Text)
			}
			return m, nil
		}

		if m.inPostDetail {
			return m.updatePostDetail(msg)
		}

		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

		if m.activeTab == postsTab && m.postsFiltering {
			return m.updatePosts(msg)
		}

		switch msg.String() {
		case "tab":
			m.activeTab = (m.activeTab + 1) % numTabs
			m = m.ensureAbout()
			return m, nil
		case "shift+tab":
			m.activeTab = (m.activeTab + numTabs - 1) % numTabs
			m = m.ensureAbout()
			return m, nil
		case "1":
			m.activeTab = homeTab
			return m, nil
		case "2":
			m.activeTab = aboutTab
			m = m.ensureAbout()
			return m, nil
		case "3":
			m.activeTab = postsTab
			return m, nil
		case "t":
			if m.theme == themeDark {
				m.theme = themeLight
			} else {
				m.theme = themeDark
			}
			if m.aboutOK {
				m = m.initAbout()
			}
			return m, nil
		case "q":
			return m, tea.Quit
		}

		switch m.activeTab {
		case homeTab:
			return m.updateHome(msg)
		case aboutTab:
			var cmd tea.Cmd
			m.aboutVP, cmd = m.aboutVP.Update(msg)
			return m, cmd
		case postsTab:
			return m.updatePosts(msg)
		}
	}

	return m, nil
}

func (m Model) updateHome(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	links := m.cfg.Content.Links
	recentCount := m.recentPostsLimit()
	hasLinks := len(links) > 0
	hasPosts := recentCount > 0

	switch msg.String() {
	case "h", "left":
		if m.homeSection == homeSectionLinks && m.homeLinkIdx > 0 {
			m.homeLinkIdx--
		}
	case "l", "right":
		if m.homeSection == homeSectionLinks && m.homeLinkIdx < len(links)-1 {
			m.homeLinkIdx++
		}
	case "j", "down":
		if m.homeSection == homeSectionLinks && hasPosts {
			m.homeSection = homeSectionPosts
			m.homePostIdx = 0
		} else if m.homeSection == homeSectionPosts && m.homePostIdx < recentCount-1 {
			m.homePostIdx++
		}
	case "k", "up":
		if m.homeSection == homeSectionPosts && m.homePostIdx > 0 {
			m.homePostIdx--
		} else if m.homeSection == homeSectionPosts && m.homePostIdx == 0 && hasLinks {
			m.homeSection = homeSectionLinks
		}
	case "enter":
		if m.homeSection == homeSectionLinks && len(links) > 0 && m.homeLinkIdx < len(links) {
			url := links[m.homeLinkIdx].URL
			m, cmd := m.setStatus("Copied to clipboard!")
			return m, tea.Batch(cmd, tea.Printf("%s", ansi.SetSystemClipboard(url)))
		}
		if m.homeSection == homeSectionPosts && m.homePostIdx < recentCount {
			return m.openPost(m.posts[m.homePostIdx])
		}
	}

	return m, nil
}

func (m Model) filteredPosts() []content.Post {
	var filtered []content.Post
	query := strings.ToLower(m.postsFilter.Value())
	for _, p := range m.posts {
		if query == "" || strings.Contains(strings.ToLower(p.Title), query) || strings.Contains(strings.ToLower(p.Description), query) {
			filtered = append(filtered, p)
		}
	}
	if m.postsSort == sortOldest {
		sort.Slice(filtered, func(i, j int) bool {
			return filtered[i].Date.Before(filtered[j].Date)
		})
	}
	return filtered
}

func (m Model) postsBodyHeight() int {
	// viewport height minus info line + blank line
	h := m.viewportHeight() - 2
	if h < 1 {
		h = 1
	}
	return h
}

func (m Model) postsVisibleCount() int {
	// Each post takes ~3 lines (date/title + description + blank line)
	h := m.postsBodyHeight()
	n := h / 3
	if n < 1 {
		n = 1
	}
	return n
}

func (m Model) updatePosts(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	filtered := m.filteredPosts()
	count := len(filtered)

	if m.postsFiltering {
		switch msg.String() {
		case "enter", "esc":
			m.postsFiltering = false
			m.postsFilter.Blur()
			if msg.String() == "esc" {
				m.postsFilter.SetValue("")
			}
			m.postsIdx = 0
			m.postsOffset = 0
			return m, nil
		default:
			var cmd tea.Cmd
			m.postsFilter, cmd = m.postsFilter.Update(msg)
			m.postsIdx = 0
			m.postsOffset = 0
			return m, cmd
		}
	}

	switch msg.String() {
	case "j", "down":
		if m.postsIdx < count-1 {
			m.postsIdx++
			visible := m.postsVisibleCount()
			if m.postsIdx >= m.postsOffset+visible {
				m.postsOffset = m.postsIdx - visible + 1
			}
		}
	case "k", "up":
		if m.postsIdx > 0 {
			m.postsIdx--
			if m.postsIdx < m.postsOffset {
				m.postsOffset = m.postsIdx
			}
		}
	case "/":
		m.postsFiltering = true
		m.postsFilter.Focus()
		return m, textinput.Blink
	case "s":
		if m.postsSort == sortNewest {
			m.postsSort = sortOldest
		} else {
			m.postsSort = sortNewest
		}
		m.postsIdx = 0
		m.postsOffset = 0
	case "enter":
		if count > 0 && m.postsIdx < count {
			return m.openPost(filtered[m.postsIdx])
		}
	}

	return m, nil
}

func (m Model) updatePostDetail(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc":
		m.inPostDetail = false
		m.activeTab = m.returnTab
		return m, nil
	case "ctrl+c":
		return m, tea.Quit
	}

	var cmd tea.Cmd
	m.postVP, cmd = m.postVP.Update(msg)
	return m, cmd
}

func (m Model) openPost(p content.Post) (Model, tea.Cmd) {
	cw := m.contentWidth()
	rendered, err := content.RenderMarkdownWithImages(
		p.Body, cw-4, m.theme.String(), m.imageMode,
		m.theme == themeDark, m.cache, m.cfg.SSH.Images.MaxSize,
		m.cfg.SSH.Images.FetchTimeout.Duration, m.cfg.SSH.Images.MaxAsciiWidth,
	)
	if err != nil {
		rendered = p.Body
	}

	vpH := m.viewportHeight()
	vp := viewport.New(viewport.WithWidth(m.contentWidth()), viewport.WithHeight(vpH))
	vp.SetContent(rendered)
	m.postVP = vp
	m.currPost = &p
	m.inPostDetail = true
	m.returnTab = m.activeTab
	return m, nil
}

func (m Model) ensureAbout() Model {
	if m.activeTab == aboutTab && !m.aboutOK && m.ready {
		return m.initAbout()
	}
	return m
}

func (m Model) initAbout() Model {
	cw := m.contentWidth()
	rendered, err := content.RenderMarkdown(m.aboutRaw, cw-4, m.theme.String())
	if err != nil {
		rendered = m.aboutRaw
	}
	vpH := m.viewportHeight()
	m.aboutVP = viewport.New(viewport.WithWidth(cw), viewport.WithHeight(vpH))
	m.aboutVP.SetContent(rendered)
	m.aboutOK = true
	return m
}

func (m Model) recentPostsLimit() int {
	n := m.cfg.Content.RecentPostsLimit
	if n > len(m.posts) {
		n = len(m.posts)
	}
	return n
}

// View

func (m Model) View() tea.View {
	v := tea.NewView("")
	v.AltScreen = true

	if !m.ready {
		v.SetContent("Loading...")
		return v
	}

	if !m.splash.done {
		v.SetContent(m.viewSplash())
		return v
	}

	cw := m.contentWidth()
	tabBar := m.renderTabBar(cw)

	var body, footer string
	if m.inPostDetail {
		body, footer = m.viewPostDetail(cw)
	} else {
		switch m.activeTab {
		case homeTab:
			body, footer = m.viewHome(cw)
		case aboutTab:
			body, footer = m.viewAbout(cw)
		case postsTab:
			body, footer = m.viewPosts(cw)
		}
	}

	tabBarH := lipgloss.Height(tabBar)
	footerH := lipgloss.Height(footer)
	bodyH := m.height - tabBarH - footerH
	if bodyH < 1 {
		bodyH = 1
	}

	styledBody := lipgloss.NewStyle().
		Height(bodyH).
		Render(body)

	content := lipgloss.JoinVertical(lipgloss.Left, tabBar, styledBody, footer)
	placed := lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Top,
		lipgloss.NewStyle().MaxWidth(cw).Render(content))

	v.SetContent(placed)
	return v
}

func (m Model) renderTabBar(contentWidth int) string {
	accent := m.accentColor()
	dim := m.dimColor()

	activeTab := m.activeTab
	if m.inPostDetail {
		activeTab = m.returnTab
	}

	var tabs []string
	for i, name := range tabNames {
		if tab(i) == activeTab {
			s := lipgloss.NewStyle().Bold(true).Foreground(accent).Padding(0, 2)
			tabs = append(tabs, s.Render(name))
		} else {
			s := lipgloss.NewStyle().Foreground(dim).Padding(0, 2)
			tabs = append(tabs, s.Render(name))
		}
	}

	bar := lipgloss.JoinHorizontal(lipgloss.Top, tabs...)
	sep := lipgloss.NewStyle().Foreground(dim).Render(strings.Repeat("─", contentWidth))
	return bar + "\n" + sep
}

func (m Model) viewHome(maxW int) (string, string) {
	var b strings.Builder
	accent := m.accentColor()
	dim := m.dimColor()
	links := m.cfg.Content.Links

	// Title
	title := lipgloss.NewStyle().Bold(true).Foreground(accent).Render(m.cfg.Content.Name)
	b.WriteString(lipgloss.NewStyle().PaddingTop(1).Render(title) + "\n")
	if m.cfg.Content.Subtitle != "" {
		b.WriteString(lipgloss.NewStyle().Foreground(dim).Render(m.cfg.Content.Subtitle) + "\n")
	}

	// Links
	if len(links) > 0 {
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Bold(true).Render("Find me on") + "\n")

		for i, link := range links {
			var s lipgloss.Style
			if m.homeSection == homeSectionLinks && i == m.homeLinkIdx {
				s = lipgloss.NewStyle().Bold(true).Foreground(accent)
			} else {
				s = lipgloss.NewStyle().Foreground(dim)
			}
			b.WriteString(s.Padding(0, 1).Render(link.Name))
		}
		b.WriteString("\n")
	}

	// Recent posts
	recentCount := m.recentPostsLimit()
	if recentCount > 0 {
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Bold(true).Render("Recent Posts") + "\n")

		for i := 0; i < recentCount; i++ {
			p := m.posts[i]
			date := lipgloss.NewStyle().Foreground(dim).Render(p.Date.Format("Jan 2006"))

			if m.homeSection == homeSectionPosts && i == m.homePostIdx {
				title := lipgloss.NewStyle().Bold(true).Foreground(accent).Render(p.Title)
				b.WriteString("> " + date + " " + title + "\n")
			} else {
				title := lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Render(p.Title)
				b.WriteString("  " + date + " " + title + "\n")
			}
		}
	}

	// Footer
	sep := lipgloss.NewStyle().Foreground(dim).Render(strings.Repeat("─", maxW))
	helpText := lipgloss.NewStyle().Foreground(dim).Render("tab switch · t theme · enter select · q quit")
	var urlText string
	if m.status != "" {
		urlText = lipgloss.NewStyle().Foreground(accent).Render(m.status)
	} else if m.homeSection == homeSectionLinks && len(links) > 0 && m.homeLinkIdx < len(links) {
		urlText = lipgloss.NewStyle().Foreground(dim).Render("↳ " + links[m.homeLinkIdx].URL)
	}
	gap := maxW - lipgloss.Width(helpText) - lipgloss.Width(urlText)
	if gap < 2 {
		gap = 2
	}
	footer := sep + "\n" + helpText + strings.Repeat(" ", gap) + urlText

	return b.String(), footer
}

func (m Model) viewAbout(maxW int) (string, string) {
	dim := m.dimColor()

	var body string
	if !m.aboutOK {
		body = "\n  No about page configured."
	} else {
		body = m.aboutVP.View()
	}

	sep := lipgloss.NewStyle().Foreground(dim).Render(strings.Repeat("─", maxW))
	helpText := lipgloss.NewStyle().Foreground(dim).Render("tab switch · t theme · q quit")
	footer := sep + "\n" + helpText

	return body, footer
}

func (m Model) viewPosts(maxW int) (string, string) {
	var b strings.Builder
	accent := m.accentColor()
	dim := m.dimColor()
	filtered := m.filteredPosts()
	count := len(filtered)
	dateCol := 12 // width for date column e.g. "Jan 2006  "

	// Info line: sort + count + filter (all on one line)
	sortLabel := "newest first"
	if m.postsSort == sortOldest {
		sortLabel = "oldest first"
	}
	info := sortLabel + " · " + fmt.Sprintf("%d posts", count)
	if m.postsFiltering {
		info += " · " + m.postsFilter.View()
	} else if m.postsFilter.Value() != "" {
		info += " · filter: " + m.postsFilter.Value()
	}
	if !m.postsFiltering {
		info = lipgloss.NewStyle().Foreground(dim).Render(info)
	}
	b.WriteString(info + "\n\n")

	// Post list
	visible := m.postsVisibleCount()
	end := m.postsOffset + visible
	if end > count {
		end = count
	}

	titleWidth := maxW - dateCol - 4 // 4 for caret + spacing
	if titleWidth < 20 {
		titleWidth = 20
	}

	for i := m.postsOffset; i < end; i++ {
		p := filtered[i]
		date := p.Date.Format("Jan 2006")
		datePad := strings.Repeat(" ", dateCol-len(date))

		if i == m.postsIdx {
			dateStr := lipgloss.NewStyle().Foreground(dim).Render(date + datePad)
			title := lipgloss.NewStyle().Bold(true).Foreground(accent).Render(p.Title)
			b.WriteString("> " + dateStr + title + "\n")
			if p.Description != "" {
				desc := lipgloss.NewStyle().Foreground(dim).Width(titleWidth).Render(p.Description)
				pad := strings.Repeat(" ", dateCol+2)
				for j, line := range strings.Split(desc, "\n") {
					if j == 0 {
						b.WriteString(pad + line + "\n")
					} else if strings.TrimSpace(line) != "" {
						b.WriteString(pad + line + "\n")
					}
				}
			}
		} else {
			dateStr := lipgloss.NewStyle().Foreground(dim).Render(date + datePad)
			title := lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Render(p.Title)
			b.WriteString("  " + dateStr + title + "\n")
			if p.Description != "" {
				desc := lipgloss.NewStyle().Foreground(dim).Width(titleWidth).Render(p.Description)
				pad := strings.Repeat(" ", dateCol+2)
				for j, line := range strings.Split(desc, "\n") {
					if j == 0 {
						b.WriteString(pad + line + "\n")
					} else if strings.TrimSpace(line) != "" {
						b.WriteString(pad + line + "\n")
					}
				}
			}
		}
		b.WriteString("\n")
	}

	// Footer
	sep := lipgloss.NewStyle().Foreground(dim).Render(strings.Repeat("─", maxW))
	helpText := lipgloss.NewStyle().Foreground(dim).Render("tab switch · t theme · / filter · s sort · enter select · q quit")
	footer := sep + "\n" + helpText

	return b.String(), footer
}

func (m Model) viewPostDetail(maxW int) (string, string) {
	dim := m.dimColor()
	sep := lipgloss.NewStyle().Foreground(dim).Render(strings.Repeat("─", maxW))
	helpText := lipgloss.NewStyle().Foreground(dim).
		Render(fmt.Sprintf("esc/q back · %3.f%% · %s", m.postVP.ScrollPercent()*100, m.theme))
	footer := sep + "\n" + helpText
	return m.postVP.View(), footer
}

// Theme colors

func (m Model) accentColor() color.Color {
	if m.theme == themeLight {
		return lipgloss.Color("63")
	}
	return lipgloss.Color("170")
}

func (m Model) dimColor() color.Color {
	if m.theme == themeLight {
		return lipgloss.Color("245")
	}
	return lipgloss.Color("241")
}
