package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
)

// performance_test.go is the permanent evidence for ticket-07-04 (fluidity / low
// consumption). It cannot measure wall-clock CPU in a unit test, so it pins the
// structural properties that GUARANTEE fluidity:
//   - slow work is deferred to a tea.Cmd, never run inline in Update/View, so the
//     loop is never blocked and input stays responsive during loading (R1/R2);
//   - the model schedules no recurring command at idle, so CPU is ~0 with no input
//     (R3) — there are no spinners/tickers to keep firing;
//   - the Glamour renderer cache returns correct output and is reused per width and
//     bounded, so repeated navigation does not grow memory (R4/R5).
// Wall-clock startup is measured by BenchmarkStartup below (RNF-1 evidence).

// TestInputNotBlockedDuringLoad covers R1/R2 (Check-3/8): while an area is still
// loading (its async result has NOT arrived), the model still services input —
// back (esc) and the help toggle (?) — instead of freezing. The work itself lives
// in a tea.Cmd, so Update returns immediately and remains responsive.
func TestInputNotBlockedDuringLoad(t *testing.T) {
	m := sizedModel(t)

	// Enter an area: this returns a load COMMAND (deferred work), and the area is in
	// the loading state because no areaLoadedMsg has been delivered yet.
	m, cmd := enterAreaByIndex(t, m, indexOf(areaPrompts))
	if cmd == nil {
		t.Fatal("entering an unloaded area should return a load command (deferred work)")
	}
	if !m.areas[areaPrompts].loading {
		t.Fatal("area should be in the loading state while the command is pending")
	}
	if !strings.Contains(m.View(), "Loading") {
		t.Errorf("a loading area should show a loading state, got:\n%s", m.View())
	}

	// While still loading, the help toggle is serviced.
	hm := update(m, "?")
	if !hm.help.ShowAll {
		t.Error("'?' should toggle help even while the area is loading (input not blocked)")
	}

	// While still loading, back (esc) returns to the root — the user can always leave.
	bm := update(m, "esc")
	if bm.current() != routeRoot {
		t.Error("'esc' should return to the root even while loading (cancel/back works)")
	}
}

// TestNoRecurringCommandAtIdle covers R3: once settled (area loaded), an unrelated
// no-op message produces NO command, so the model never schedules recurring work
// (no tick/poll) when there is no input. This is what keeps idle CPU ~0.
func TestNoRecurringCommandAtIdle(t *testing.T) {
	// Root menu is the purest idle state: Init returns no command, and a stray
	// message must not start one.
	m := sizedModel(t)
	if cmd := m.Init(); cmd != nil {
		t.Error("Init should schedule no command (lazy, idle-friendly startup)")
	}
	if _, cmd := m.Update(idleNopMsg{}); cmd != nil {
		t.Error("a no-op message at the root must not schedule a command (no idle polling)")
	}

	// Settled area: deliver the load, then a no-op must still produce no command.
	a := loadedAgentsArea(t)
	if _, cmd := a.Update(idleNopMsg{}); cmd != nil {
		t.Error("a no-op message in a settled area must not schedule a command (no idle polling)")
	}

	// Settled sub-screen (a loaded document): a no-op must not schedule a command.
	s := sizedModel(t)
	s, _ = enterAreaByIndex(t, s, indexOf(areaPrompts))
	s = deliverAreaLoaded(s, areaPrompts, []areaItem{{key: "p", label: "p", opens: true}})
	s = update(s, "enter")
	s = deliverSubLoaded(s, areaPrompts, "p", "rendered body")
	if _, cmd := s.Update(idleNopMsg{}); cmd != nil {
		t.Error("a no-op message on a settled sub-screen must not schedule a command")
	}
}

// idleNopMsg is an arbitrary message type the model does not handle, used to prove
// that an unhandled tick-like message does not cause the model to schedule recurring
// work.
type idleNopMsg struct{}

// TestSettledAreaLoadIsOneShot covers R3/R6: delivering an area's load result
// produces no follow-up command (the load is one-shot, not a repeating refresh).
func TestSettledAreaLoadIsOneShot(t *testing.T) {
	m := sizedModel(t)
	m, _ = enterAreaByIndex(t, m, indexOf(areaPrompts))
	_, cmd := m.Update(areaLoadedMsg{id: areaPrompts, items: []areaItem{{key: "p", label: "p"}}})
	if cmd != nil {
		t.Error("handling an areaLoadedMsg should not schedule further work (one-shot load)")
	}
}

// TestMarkdownCacheReusesRenderer covers R4: the Glamour renderer is cached per
// width, so two renders at the same width reuse one renderer (no per-render
// reallocation). We assert reuse by identity of the cached pointer.
func TestMarkdownCacheReusesRenderer(t *testing.T) {
	th := defaultTheme()
	resetRendererCache()

	r1 := th.renderer(72)
	r2 := th.renderer(72)
	if r1 == nil || r2 == nil {
		t.Fatal("renderer construction failed")
	}
	if r1 != r2 {
		t.Error("the same width should reuse the cached renderer (no reallocation)")
	}

	// A different width builds (and caches) a distinct renderer.
	r3 := th.renderer(40)
	if r3 == r1 {
		t.Error("a different width should use a different renderer")
	}
}

// TestMarkdownCacheBounded covers R4/R5: the cache never grows past its bound — many
// distinct widths must not accumulate unboundedly.
func TestMarkdownCacheBounded(t *testing.T) {
	th := defaultTheme()
	resetRendererCache()

	for w := 20; w < 20+maxCachedRenderers*4; w++ {
		_ = th.renderer(w)
	}

	rendererCache.mu.Lock()
	size := len(rendererCache.m)
	rendererCache.mu.Unlock()

	if size > maxCachedRenderers {
		t.Errorf("renderer cache grew to %d, exceeding the bound %d", size, maxCachedRenderers)
	}
}

// TestMarkdownRenderCorrectThroughCache covers R4: caching must not change output —
// the cached renderer produces the same correct, formatted result.
func TestMarkdownRenderCorrectThroughCache(t *testing.T) {
	th := defaultTheme()
	resetRendererCache()

	out1 := th.renderMarkdownWidth("# Title\n\nbody **x**", 72)
	out2 := th.renderMarkdownWidth("# Title\n\nbody **x**", 72)
	if out1 != out2 {
		t.Error("cached renders of the same input/width should be identical")
	}
	if !strings.Contains(visibleText(out1), "Title") {
		t.Errorf("rendered output should contain the heading text, got:\n%s", visibleText(out1))
	}
}

// resetRendererCache clears the package renderer cache so cache tests start from a
// known state regardless of order.
func resetRendererCache() {
	rendererCache.mu.Lock()
	rendererCache.m = map[int]*glamour.TermRenderer{}
	rendererCache.mu.Unlock()
}

// --- benchmarks (compile in the normal build; run only with -bench) ---------

// BenchmarkStartup measures the cost of constructing the model and producing the
// first frame — the work between launch and an interactive root screen (RNF-1
// evidence). It does no disk I/O (Init is lazy), so this is the true startup cost.
func BenchmarkStartup(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		m := New(".")
		// Give it a size and render the first frame, as the runtime would.
		updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 32})
		m = updated.(Model)
		_ = m.View()
	}
}

// BenchmarkRenderMarkdownCached measures a markdown render reusing the cached
// renderer (the steady-state cost once a width is warm).
func BenchmarkRenderMarkdownCached(b *testing.B) {
	th := defaultTheme()
	resetRendererCache()
	doc := strings.Repeat("# Heading\n\nSome **bold** and *italic* text with a list:\n\n- one\n- two\n\n```go\nfmt.Println(\"hi\")\n```\n\n", 20)
	th.renderMarkdownWidth(doc, 80) // warm the cache

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = th.renderMarkdownWidth(doc, 80)
	}
}

// BenchmarkRenderMarkdownColdRenderer measures building a fresh renderer plus a
// render every iteration (the pre-cache behavior), to show the cache's effect.
func BenchmarkRenderMarkdownColdRenderer(b *testing.B) {
	th := defaultTheme()
	doc := strings.Repeat("# Heading\n\nSome **bold** and *italic* text with a list:\n\n- one\n- two\n\n```go\nfmt.Println(\"hi\")\n```\n\n", 20)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resetRendererCache() // force a rebuild each iteration
		_ = th.renderMarkdownWidth(doc, 80)
	}
}

// BenchmarkRendererLookupCached isolates the cache's actual saving: the cost the
// cache eliminates on every repeated navigation to a document at a known width is a
// renderer CONSTRUCTION. This measures a warm cache lookup (what we now do).
func BenchmarkRendererLookupCached(b *testing.B) {
	th := defaultTheme()
	resetRendererCache()
	th.renderer(80) // warm

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = th.renderer(80)
	}
}

// BenchmarkRendererConstruct measures a renderer construction (what the cache
// avoids on repeat). The gap between this and BenchmarkRendererLookupCached is the
// per-navigation saving for re-opening documents at the same terminal width.
func BenchmarkRendererConstruct(b *testing.B) {
	th := defaultTheme()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resetRendererCache()
		_ = th.renderer(80)
	}
}
