// Package xango is the runtime for xango components: Node trees, HTML
// rendering and event bindings.
//
// Render and RenderIndented escape text and attribute values but do not
// interpret them. User-supplied URLs placed in href, src, formaction or any
// other URL-bearing attribute MUST be validated by the caller (for example
// against an allow-list of schemes such as http and https) before they reach
// an Attr, because escaping does not neutralise schemes like javascript:.
//
// Tag and attribute names MUST match ^[A-Za-z][A-Za-z0-9:_.-]*$; rendering a
// node with any other name returns an error. Raw content is emitted verbatim
// and MUST be trusted.
package xango
