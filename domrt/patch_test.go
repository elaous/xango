//go:build js && wasm

package domrt

import (
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
