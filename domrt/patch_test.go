//go:build js && wasm

package domrt

import (
	"fmt"
	"math/rand"
	"strings"
	"syscall/js"
	"testing"

	"github.com/elaous/xango"
)

func newContainer(t *testing.T) js.Value {
	t.Helper()
	container := document().Call("createElement", "div")
	document().Get("body").Call("appendChild", container)
	t.Cleanup(func() { document().Get("body").Call("removeChild", container) })
	return container
}

func children(label string, textareaText string, middle ...xango.Node) xango.Node {
	return xango.El("li", nil,
		xango.El("span", nil, xango.Text(label)),
		xango.Frag(middle...),
		xango.El("textarea", nil, xango.Text(textareaText)),
	)
}

func recordMoves(container js.Value) (stop func() []string) {
	observer := js.Global().Get("MutationObserver").New(js.FuncOf(func(js.Value, []js.Value) any { return nil }))
	observer.Call("observe", container, map[string]any{"childList": true, "subtree": true})
	return func() []string {
		var moves []string
		records := observer.Call("takeRecords")
		observer.Call("disconnect")
		for i := 0; i < records.Length(); i++ {
			for _, kind := range []string{"addedNodes", "removedNodes"} {
				list := records.Index(i).Get(kind)
				for j := 0; j < list.Length(); j++ {
					moves = append(moves, kind+":"+list.Index(j).Get("nodeName").String())
				}
			}
		}
		return moves
	}
}

func childNames(parent js.Value) []string {
	var names []string
	kids := parent.Get("childNodes")
	for i := 0; i < kids.Length(); i++ {
		names = append(names, kids.Index(i).Get("nodeName").String())
	}
	return names
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestEmptyFragmentKeepsSiblingsInPlace(t *testing.T) {
	container := newContainer(t)
	root := Mount(container, xango.El("ul", nil, children("a", "x")))
	textarea := container.Call("querySelector", "textarea")
	textarea.Call("focus")

	stop := recordMoves(container)
	root.Update(xango.El("ul", nil, children("b", "x")))
	moves := stop()

	if len(moves) != 0 {
		t.Fatalf("unexpected DOM moves: %v", moves)
	}
	if !container.Call("querySelector", "textarea").Equal(textarea) {
		t.Fatal("textarea node was replaced")
	}
	if !document().Get("activeElement").Equal(textarea) {
		t.Fatal("textarea lost focus")
	}
	if got := container.Call("querySelector", "span").Get("textContent").String(); got != "b" {
		t.Fatalf("span text = %q, want b", got)
	}
}

func TestEmptyFragmentGainsChildren(t *testing.T) {
	container := newContainer(t)
	root := Mount(container, xango.El("ul", nil, children("a", "x")))
	textarea := container.Call("querySelector", "textarea")
	textarea.Call("focus")

	root.Update(xango.El("ul", nil, children("a", "x", xango.El("i", nil), xango.El("b", nil))))

	li := container.Call("querySelector", "li")
	if got, want := childNames(li), []string{"SPAN", "I", "B", "TEXTAREA"}; !equalStrings(got, want) {
		t.Fatalf("children = %v, want %v", got, want)
	}
	if !li.Call("querySelector", "textarea").Equal(textarea) {
		t.Fatal("textarea node was replaced")
	}
	if !document().Get("activeElement").Equal(textarea) {
		t.Fatal("textarea lost focus")
	}
}

func TestFragmentLosesChildren(t *testing.T) {
	container := newContainer(t)
	root := Mount(container, xango.El("ul", nil, children("a", "x", xango.El("i", nil), xango.El("b", nil))))
	textarea := container.Call("querySelector", "textarea")
	textarea.Call("focus")

	stop := recordMoves(container)
	root.Update(xango.El("ul", nil, children("a", "x")))
	moves := stop()

	li := container.Call("querySelector", "li")
	if got, want := childNames(li), []string{"SPAN", "TEXTAREA"}; !equalStrings(got, want) {
		t.Fatalf("children = %v, want %v", got, want)
	}
	if want := []string{"removedNodes:I", "removedNodes:B"}; !equalStrings(moves, want) {
		t.Fatalf("DOM moves = %v, want %v", moves, want)
	}
	if !document().Get("activeElement").Equal(textarea) {
		t.Fatal("textarea lost focus")
	}
}

func TestNestedFragmentsKeepOrder(t *testing.T) {
	container := newContainer(t)
	tree := func(inner ...xango.Node) xango.Node {
		return xango.El("div", nil,
			xango.El("p", nil),
			xango.Frag(xango.Frag(inner...), xango.Frag()),
			xango.El("textarea", nil),
		)
	}
	root := Mount(container, tree())
	textarea := container.Call("querySelector", "textarea")
	textarea.Call("focus")

	root.Update(tree(xango.El("i", nil)))
	div := container.Call("querySelector", "div")
	if got, want := childNames(div), []string{"P", "I", "TEXTAREA"}; !equalStrings(got, want) {
		t.Fatalf("children = %v, want %v", got, want)
	}
	if !document().Get("activeElement").Equal(textarea) {
		t.Fatal("textarea lost focus")
	}

	root.Update(tree())
	if got, want := childNames(div), []string{"P", "TEXTAREA"}; !equalStrings(got, want) {
		t.Fatalf("children = %v, want %v", got, want)
	}
}

func TestTrailingFragmentInsideFragmentKeepsAnchor(t *testing.T) {
	container := newContainer(t)
	tree := func(tail ...xango.Node) xango.Node {
		return xango.El("div", nil,
			xango.Frag(xango.El("p", nil), xango.Frag(tail...)),
			xango.El("textarea", nil),
		)
	}
	root := Mount(container, tree())
	root.Update(tree(xango.El("i", nil)))

	div := container.Call("querySelector", "div")
	if got, want := childNames(div), []string{"P", "I", "TEXTAREA"}; !equalStrings(got, want) {
		t.Fatalf("children = %v, want %v", got, want)
	}
}

func innerHTML(container js.Value) string { return container.Get("innerHTML").String() }

func updateRecovering(root *Root, next xango.Node) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	root.Update(next)
	return nil
}

func expectHTML(t *testing.T, container js.Value, want string) {
	t.Helper()
	if got := innerHTML(container); got != want {
		t.Fatalf("html = %s, want %s", got, want)
	}
}

func TestKeyedReplacedSiblingStaleAnchor(t *testing.T) {
	container := newContainer(t)
	root := Mount(container, xango.El("div", nil, xango.Frag().WithKey("a"), xango.El("p", nil).WithKey("b")))
	next := xango.El("div", nil, xango.El("i", nil).WithKey("b"), xango.Frag(xango.El("u", nil)).WithKey("a"))
	if err := updateRecovering(root, next); err != nil {
		t.Fatal(err)
	}
	expectHTML(t, container, "<div><i></i><u></u></div>")
}

func TestKeyedReplacedSiblingKeepsFocusBeyondStaleAnchor(t *testing.T) {
	container := newContainer(t)
	root := Mount(container, xango.El("div", nil,
		xango.Frag().WithKey("a"), xango.El("p", nil).WithKey("b"), xango.El("textarea", nil).WithKey("t")))
	textarea := container.Call("querySelector", "textarea")
	textarea.Call("focus")
	root.Update(xango.El("div", nil,
		xango.El("i", nil).WithKey("b"), xango.Frag(xango.El("u", nil)).WithKey("a"), xango.El("textarea", nil).WithKey("t")))
	expectHTML(t, container, "<div><i></i><u></u><textarea></textarea></div>")
	if !document().Get("activeElement").Equal(textarea) {
		t.Fatal("textarea lost focus")
	}
}

func TestConsecutiveEmptyFragments(t *testing.T) {
	container := newContainer(t)
	tree := func(filled int) xango.Node {
		kids := []xango.Node{xango.El("a", nil)}
		for i := 0; i < 3; i++ {
			if i == filled {
				kids = append(kids, xango.Frag(xango.El("u", nil)))
			} else {
				kids = append(kids, xango.Frag())
			}
		}
		return xango.El("div", nil, append(kids, xango.El("textarea", nil))...)
	}
	root := Mount(container, tree(-1))
	textarea := container.Call("querySelector", "textarea")
	textarea.Call("focus")
	for filled, want := range map[int]string{
		1:  "<div><a></a><u></u><textarea></textarea></div>",
		2:  "<div><a></a><u></u><textarea></textarea></div>",
		0:  "<div><a></a><u></u><textarea></textarea></div>",
		-1: "<div><a></a><textarea></textarea></div>",
	} {
		root.Update(tree(filled))
		expectHTML(t, container, want)
	}
	if !document().Get("activeElement").Equal(textarea) {
		t.Fatal("textarea lost focus")
	}
}

func TestEmptyFragmentAtStart(t *testing.T) {
	container := newContainer(t)
	tree := func(inner ...xango.Node) xango.Node {
		return xango.El("div", nil, xango.Frag(inner...), xango.El("textarea", nil))
	}
	root := Mount(container, tree())
	root.Update(tree(xango.El("i", nil)))
	expectHTML(t, container, "<div><i></i><textarea></textarea></div>")
	root.Update(tree())
	expectHTML(t, container, "<div><textarea></textarea></div>")
}

func TestElementRemovedBeforeEmptyFragment(t *testing.T) {
	container := newContainer(t)
	root := Mount(container, xango.El("div", nil,
		xango.El("a", nil).WithKey("a"), xango.El("b", nil).WithKey("b"),
		xango.Frag().WithKey("f"), xango.El("textarea", nil).WithKey("t")))
	textarea := container.Call("querySelector", "textarea")
	textarea.Call("focus")
	root.Update(xango.El("div", nil,
		xango.El("a", nil).WithKey("a"),
		xango.Frag(xango.El("u", nil)).WithKey("f"), xango.El("textarea", nil).WithKey("t")))
	expectHTML(t, container, "<div><a></a><u></u><textarea></textarea></div>")
	if !document().Get("activeElement").Equal(textarea) {
		t.Fatal("textarea lost focus")
	}
}

func TestEmptyFragmentRoundTrip(t *testing.T) {
	container := newContainer(t)
	tree := func(inner ...xango.Node) xango.Node {
		return xango.El("div", nil, xango.El("a", nil), xango.Frag(inner...), xango.El("textarea", nil))
	}
	root := Mount(container, tree())
	textarea := container.Call("querySelector", "textarea")
	textarea.Call("focus")
	root.Update(tree(xango.El("i", nil)))
	expectHTML(t, container, "<div><a></a><i></i><textarea></textarea></div>")
	root.Update(tree())
	expectHTML(t, container, "<div><a></a><textarea></textarea></div>")
	if !document().Get("activeElement").Equal(textarea) {
		t.Fatal("textarea lost focus")
	}
}

func TestEmptyFragmentReplacedByElement(t *testing.T) {
	container := newContainer(t)
	root := Mount(container, xango.El("div", nil, xango.El("a", nil), xango.Frag(), xango.El("c", nil)))
	root.Update(xango.El("div", nil, xango.El("a", nil), xango.El("b", nil), xango.El("c", nil)))
	expectHTML(t, container, "<div><a></a><b></b><c></c></div>")
}

func describeNode(n xango.Node) string {
	parts := make([]string, 0, len(n.Children))
	for _, c := range n.Children {
		parts = append(parts, describeNode(c))
	}
	return fmt.Sprintf("%d:%s#%s(%s)[%s]", n.Kind, n.Tag, n.Key, n.Text, strings.Join(parts, ","))
}

func randomChildren(r *rand.Rand, depth int, keyed bool) []xango.Node {
	var out []xango.Node
	used := map[string]bool{}
	for i, count := 0, r.Intn(5); i < count; i++ {
		var n xango.Node
		switch k := r.Intn(8); {
		case k == 0:
			n = xango.Text(fmt.Sprint("t", r.Intn(3)))
		case k <= 3 && depth > 0:
			n = xango.El([]string{"p", "i"}[r.Intn(2)], nil, randomChildren(r, depth-1, keyed)...)
		case depth > 0 && r.Intn(3) > 0:
			n = xango.Frag(randomChildren(r, depth-1, keyed)...)
		default:
			n = xango.Frag()
		}
		if keyed && r.Intn(2) == 0 {
			if key := fmt.Sprint("k", r.Intn(6)); !used[key] {
				used[key] = true
				n = n.WithKey(key)
			}
		}
		out = append(out, n)
	}
	return out
}

func TestDifferentialAgainstFreshMount(t *testing.T) {
	for seed := int64(0); seed < 1500; seed++ {
		r := rand.New(rand.NewSource(seed))
		keyed := seed%2 == 0
		wrap := func(kids []xango.Node) xango.Node {
			if seed%3 == 0 {
				return xango.Frag(kids...)
			}
			return xango.El("div", nil, kids...)
		}
		a, b, c := wrap(randomChildren(r, 3, keyed)), wrap(randomChildren(r, 3, keyed)), wrap(randomChildren(r, 3, keyed))
		container := newContainer(t)
		root := Mount(container, a)
		for step, next := range []xango.Node{b, c, a} {
			if err := updateRecovering(root, next); err != nil {
				t.Fatalf("seed %d step %d: %v\n a=%s\n next=%s", seed, step, err, describeNode(a), describeNode(next))
			}
			fresh := newContainer(t)
			Mount(fresh, next)
			if got, want := innerHTML(container), innerHTML(fresh); got != want {
				t.Fatalf("seed %d step %d: html = %s, want %s\n a=%s\n next=%s", seed, step, got, want, describeNode(a), describeNode(next))
			}
		}
		stop := recordMoves(container)
		root.Update(a)
		if moves := stop(); len(moves) != 0 {
			t.Fatalf("seed %d: no-op update moved nodes %v\n a=%s", seed, moves, describeNode(a))
		}
	}
}

func TestManyEmptyFragmentsStayFast(t *testing.T) {
	const count = 20000
	tree := func() xango.Node {
		kids := make([]xango.Node, 0, count+1)
		for i := 0; i < count; i++ {
			kids = append(kids, xango.Frag())
		}
		return xango.El("div", nil, append(kids, xango.El("a", nil))...)
	}
	container := newContainer(t)
	root := Mount(container, tree())
	now := func() float64 { return js.Global().Get("performance").Call("now").Float() }
	start := now()
	root.Update(tree())
	if elapsed := now() - start; elapsed > 1000 {
		t.Fatalf("update took %.0fms", elapsed)
	}
}
