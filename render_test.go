package xango

import "testing"

func TestRenderBasic(t *testing.T) {
	n := El("div", []Attr{A("class", "greeting")},
		El("p", nil, Text("Hello, <world> & \"friends\"!")),
	)
	got, err := RenderString(n)
	if err != nil {
		t.Fatal(err)
	}
	want := `<div class="greeting"><p>Hello, &lt;world&gt; &amp; &#34;friends&#34;!</p></div>`
	if got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
}

func TestRenderVoidElement(t *testing.T) {
	n := El("img", []Attr{A("src", "cat.png")}, Text("should be ignored"))
	got, err := RenderString(n)
	if err != nil {
		t.Fatal(err)
	}
	want := `<img src="cat.png">`
	if got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
}

func TestRenderFragmentAndEventsAreInert(t *testing.T) {
	n := Frag(
		El("button", []Attr{On("click", func(Event) {})}, Text("Click")),
		Text("!"),
	)
	got, err := RenderString(n)
	if err != nil {
		t.Fatal(err)
	}
	want := `<button>Click</button>!`
	if got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
}

func TestRenderRaw(t *testing.T) {
	n := El("div", nil, Raw("<b>bold</b>"))
	got, err := RenderString(n)
	if err != nil {
		t.Fatal(err)
	}
	want := `<div><b>bold</b></div>`
	if got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
}

func TestRenderNameValidation(t *testing.T) {
	valid := []Node{
		El("div", nil),
		El("my-widget", []Attr{A("data-x", "1"), A("xlink:href", "#a"), A("aria.label", "x")}),
		El("linearGradient", nil),
		El("a_b", nil),
	}
	for _, n := range valid {
		if _, err := RenderString(n); err != nil {
			t.Errorf("%q: unexpected error %v", n.Tag, err)
		}
		if _, err := RenderIndented(n); err != nil {
			t.Errorf("%q: unexpected indented error %v", n.Tag, err)
		}
	}

	invalid := map[string]Node{
		"empty tag":                 El("", nil),
		"tag with space":            El("div onclick=alert(1)", nil),
		"tag with bracket":          El("div><script>", nil),
		"tag leading digit":         El("1div", nil),
		"tag leading dash":          El("-x", nil),
		"tag non-ascii":             El("dív", nil),
		"attr empty":                El("div", []Attr{A("", "v")}),
		"attr with space":           El("div", []Attr{A("a b", "v")}),
		"attr with equals":          El("div", []Attr{A("a=b", "v")}),
		"attr with quote":           El("div", []Attr{A(`a"onload="x`, "v")}),
		"attr with slash":           El("div", []Attr{A("a/b", "v")}),
		"nested invalid":            El("div", nil, El("p", []Attr{A("bad name", "v")})),
		"invalid in text container": El("p", nil, Text("x"), El("b c", nil)),
	}
	for name, n := range invalid {
		if _, err := RenderString(n); err == nil {
			t.Errorf("%s: Render accepted invalid name", name)
		}
		if _, err := RenderIndented(El("section", nil, n)); err == nil {
			t.Errorf("%s: RenderIndented accepted invalid name", name)
		}
	}
}

func TestRenderIgnoresHandlerNamesInHTML(t *testing.T) {
	n := El("button", []Attr{On("on click", func(Event) {})})
	got, err := RenderString(n)
	if err != nil || got != "<button></button>" {
		t.Errorf("got %q, %v", got, err)
	}
}

func BenchmarkRenderElement(b *testing.B) {
	n := El("div", []Attr{A("class", "x"), A("data-id", "1")}, Text("hi"))
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = RenderString(n)
	}
}
