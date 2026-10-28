// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package html

import (
	"slices"
	"strings"

	a "golang.org/x/net/html/atom"
)

func inBodyText(p *parser) {
	d := p.tok.Data
	switch n := p.oe.top(); n.DataAtom {
	case a.Pre, a.Listing:
		if n.FirstChild == nil {
			// Ignore a newline at the start of a <pre> block.
			if d != "" && d[0] == '\r' {
				d = d[1:]
			}
			if d != "" && d[0] == '\n' {
				d = d[1:]
			}
		}
	}
	d = strings.ReplaceAll(d, "\x00", "")
	if d == "" {
		return
	}
	p.reconstructActiveFormattingElements()
	p.addText(d)
	if p.framesetOK && strings.TrimLeft(d, whitespace) != "" {
		// There were non-whitespace characters inserted.
		p.framesetOK = false
	}
}

func inBodyStartTag(p *parser) bool {
	switch p.tok.DataAtom {
	case a.Form, a.Button, a.Input, a.Textarea, a.Select, a.Option, a.Optgroup:
		inBodyStartControl(p)
	case a.Html:
		if p.oe.contains(a.Template) {
			return true
		}
		copyAttributes(p.oe[0], p.tok)
	case a.Base, a.Basefont, a.Bgsound, a.Link, a.Meta, a.Noframes, a.Script, a.Style, a.Template, a.Title:
		return inHeadIM(p)
	case a.Body:
		if p.oe.contains(a.Template) {
			return true
		}
		if len(p.oe) >= 2 {
			body := p.oe[1]
			if body.Type == ElementNode && body.DataAtom == a.Body {
				p.framesetOK = false
				copyAttributes(body, p.tok)
			}
		}
	case a.Frameset:
		inBodyStartFrameset(p)
	case a.Address, a.Article, a.Aside, a.Blockquote, a.Center, a.Details, a.Dialog, a.Dir, a.Div, a.Dl, a.Fieldset, a.Figcaption, a.Figure, a.Footer, a.Header, a.Hgroup, a.Main, a.Menu, a.Nav, a.Ol, a.P, a.Search, a.Section, a.Summary, a.Ul:
		p.popUntil(buttonScope, a.P)
		p.addElement()
	case a.H1, a.H2, a.H3, a.H4, a.H5, a.H6:
		p.popUntil(buttonScope, a.P)
		switch n := p.top(); n.DataAtom {
		case a.H1, a.H2, a.H3, a.H4, a.H5, a.H6:
			p.oe.pop()
		}
		p.addElement()
	case a.Pre, a.Listing:
		p.popUntil(buttonScope, a.P)
		p.addElement()
		// The newline, if any, will be dealt with by the TextToken case.
		p.framesetOK = false
	case a.Li:
		inBodyStartListItem(p, a.Li)
	case a.Dd, a.Dt:
		inBodyStartListItem(p, a.Dd, a.Dt)
	case a.Plaintext:
		p.popUntil(buttonScope, a.P)
		p.addElement()
	case a.A:
		inBodyStartAnchor(p)
	case a.B, a.Big, a.Code, a.Em, a.Font, a.I, a.S, a.Small, a.Strike, a.Strong, a.Tt, a.U:
		p.reconstructActiveFormattingElements()
		p.addFormattingElement()
	case a.Nobr:
		p.reconstructActiveFormattingElements()
		if p.elementInScope(defaultScope, a.Nobr) {
			p.inBodyEndTagFormatting(a.Nobr, "nobr")
			p.reconstructActiveFormattingElements()
		}
		p.addFormattingElement()
	case a.Applet, a.Marquee, a.Object:
		p.reconstructActiveFormattingElements()
		p.addElement()
		p.afe = append(p.afe, &scopeMarker)
		p.framesetOK = false
	case a.Table:
		if !p.quirks {
			p.popUntil(buttonScope, a.P)
		}
		p.addElement()
		p.framesetOK = false
		p.im = inTableIM
		return true
	case a.Area, a.Br, a.Embed, a.Img, a.Keygen, a.Wbr:
		p.reconstructActiveFormattingElements()
		p.addElement()
		p.oe.pop()
		p.acknowledgeSelfClosingTag()
		p.framesetOK = false
	case a.Param, a.Source, a.Track:
		p.addElement()
		p.oe.pop()
		p.acknowledgeSelfClosingTag()
	case a.Hr:
		if p.elementInScope(buttonScope, a.P) {
			p.generateImpliedEndTags("p")
			p.popUntil(defaultScope, a.P)
		}
		if p.elementInScope(defaultScope, a.Select) {
			p.generateImpliedEndTags()
		}
		p.addElement()
		p.oe.pop()
		p.acknowledgeSelfClosingTag()
		p.framesetOK = false
	case a.Image:
		p.tok.DataAtom = a.Img
		p.tok.Data = a.Img.String()
		return false
	case a.Xmp:
		p.popUntil(buttonScope, a.P)
		p.reconstructActiveFormattingElements()
		p.framesetOK = false
		p.parseGenericRawTextElement()
	case a.Iframe:
		p.framesetOK = false
		p.parseGenericRawTextElement()
	case a.Noembed:
		p.parseGenericRawTextElement()
	case a.Noscript:
		if p.scripting {
			p.parseGenericRawTextElement()
			return true
		}
		p.reconstructActiveFormattingElements()
		p.addElement()
		// Don't let the tokenizer go into raw text mode when scripting is disabled.
		p.tokenizer.NextIsNotRawText()
	case a.Rb, a.Rtc:
		if p.elementInScope(defaultScope, a.Ruby) {
			p.generateImpliedEndTags()
		}
		p.addElement()
	case a.Rp, a.Rt:
		if p.elementInScope(defaultScope, a.Ruby) {
			p.generateImpliedEndTags("rtc")
		}
		p.addElement()
	case a.Math, a.Svg:
		inBodyStartForeign(p)
	case a.Caption, a.Col, a.Colgroup, a.Frame, a.Head, a.Tbody, a.Td, a.Tfoot, a.Th, a.Thead, a.Tr:
		// Ignore the token.
	default:
		p.reconstructActiveFormattingElements()
		p.addElement()
	}
	return true
}

func inBodyEndTag(p *parser) bool {
	switch p.tok.DataAtom {
	case a.Body:
		if p.elementInScope(defaultScope, a.Body) {
			p.im = afterBodyIM
		}
	case a.Html:
		if p.elementInScope(defaultScope, a.Body) {
			p.parseImpliedToken(EndTagToken, a.Body, a.Body.String())
			return false
		}
		return true
	case a.Address, a.Article, a.Aside, a.Blockquote, a.Button, a.Center, a.Details, a.Dialog, a.Dir, a.Div, a.Dl, a.Fieldset, a.Figcaption, a.Figure, a.Footer, a.Header, a.Hgroup, a.Listing, a.Main, a.Menu, a.Nav, a.Ol, a.Pre, a.Search, a.Section, a.Select, a.Summary, a.Ul:
		if !p.elementInScope(defaultScope, p.tok.DataAtom) {
			// Ignore the token.
			return true
		}
		p.generateImpliedEndTags()
		p.popUntil(defaultScope, p.tok.DataAtom)
	case a.Form:
		inBodyEndForm(p)
	case a.P:
		if !p.elementInScope(buttonScope, a.P) {
			p.parseImpliedToken(StartTagToken, a.P, a.P.String())
		}
		p.popUntil(buttonScope, a.P)
	case a.Li:
		p.popUntil(listItemScope, a.Li)
	case a.Dd, a.Dt:
		p.popUntil(defaultScope, p.tok.DataAtom)
	case a.H1, a.H2, a.H3, a.H4, a.H5, a.H6:
		p.popUntil(defaultScope, a.H1, a.H2, a.H3, a.H4, a.H5, a.H6)
	case a.A, a.B, a.Big, a.Code, a.Em, a.Font, a.I, a.Nobr, a.S, a.Small, a.Strike, a.Strong, a.Tt, a.U:
		p.inBodyEndTagFormatting(p.tok.DataAtom, p.tok.Data)
	case a.Applet, a.Marquee, a.Object:
		if p.popUntil(defaultScope, p.tok.DataAtom) {
			p.clearActiveFormattingElements()
		}
	case a.Br:
		p.tok.Type = StartTagToken
		return false
	case a.Template:
		return inHeadIM(p)
	default:
		p.inBodyEndTagOther(p.tok.DataAtom, p.tok.Data)
	}
	return true
}

func inBodyEOF(p *parser) bool {
	// TODO: remove this divergence from the HTML5 spec.
	if len(p.templateStack) > 0 {
		p.im = inTemplateIM
		return false
	}
	for _, e := range p.oe {
		switch e.DataAtom {
		case a.Dd, a.Dt, a.Li, a.Optgroup, a.Option, a.P, a.Rb, a.Rp, a.Rt, a.Rtc, a.Tbody, a.Td, a.Tfoot, a.Th,
			a.Thead, a.Tr, a.Body, a.Html:
		default:
			return true
		}
	}
	return true
}

func inBodyStartFrameset(p *parser) {
	if !p.framesetOK || len(p.oe) < 2 || p.oe[1].DataAtom != a.Body {
		// Ignore the token.
		return
	}
	body := p.oe[1]
	if body.Parent != nil {
		body.Parent.RemoveChild(body)
	}
	p.oe = p.oe[:1]
	p.addElement()
	p.im = inFramesetIM
}

func inBodyStartForm(p *parser) {
	if p.form != nil && !p.oe.contains(a.Template) {
		// Ignore the token
		return
	}
	p.popUntil(buttonScope, a.P)
	p.addElement()
	if !p.oe.contains(a.Template) {
		p.form = p.top()
	}
}

func inBodyStartAnchor(p *parser) {
	for i := len(p.afe) - 1; i >= 0 && p.afe[i].Type != scopeMarkerNode; i-- {
		if n := p.afe[i]; n.Type == ElementNode && n.DataAtom == a.A {
			p.inBodyEndTagFormatting(a.A, "a")
			p.oe.remove(n)
			p.afe.remove(n)
			break
		}
	}
	p.reconstructActiveFormattingElements()
	p.addFormattingElement()
}

func inBodyStartInput(p *parser) {
	if p.fragment && p.context.DataAtom == a.Select {
		// Ignore the token.
		return
	}
	p.popUntil(defaultScope, a.Select)
	p.reconstructActiveFormattingElements()
	p.addElement()
	p.oe.pop()
	p.acknowledgeSelfClosingTag()
	if p.tok.DataAtom == a.Input {
		for _, t := range p.tok.Attr {
			if t.Key == "type" {
				if strings.EqualFold(t.Val, "hidden") {
					// Skip setting framesetOK = false
					return
				}
			}
		}
	}
	p.framesetOK = false
}

func inBodyStartForeign(p *parser) {
	p.reconstructActiveFormattingElements()
	if p.tok.DataAtom == a.Math {
		adjustAttributeNames(p.tok.Attr, mathMLAttributeAdjustments)
	} else {
		adjustAttributeNames(p.tok.Attr, svgAttributeAdjustments)
	}
	adjustForeignAttributes(p.tok.Attr)
	p.addElement()
	p.top().Namespace = p.tok.Data
	if p.hasSelfClosingToken {
		p.oe.pop()
		p.acknowledgeSelfClosingTag()
	}
}

func inBodyEndForm(p *parser) {
	if p.oe.contains(a.Template) {
		i := p.indexOfElementInScope(defaultScope, a.Form)
		if i == -1 {
			// Ignore the token.
			return
		}
		p.generateImpliedEndTags()
		if p.oe[i].DataAtom != a.Form {
			// Ignore the token.
			return
		}
		p.popUntil(defaultScope, a.Form)
	} else {
		node := p.form
		p.form = nil
		i := p.indexOfElementInScope(defaultScope, a.Form)
		if node == nil || i == -1 || p.oe[i] != node {
			// Ignore the token.
			return
		}
		p.generateImpliedEndTags()
		p.oe.remove(node)
	}
}

func inBodyStartListItem(p *parser, tags ...a.Atom) {
	p.framesetOK = false
	for i := len(p.oe) - 1; i >= 0; i-- {
		node := p.oe[i]
		if slices.Contains(tags, node.DataAtom) {
			p.oe = p.oe[:i]
		} else if node.DataAtom == a.Address || node.DataAtom == a.Div || node.DataAtom == a.P || !isSpecialElement(node) {
			continue
		}
		break
	}
	p.popUntil(buttonScope, a.P)
	p.addElement()
}

func inBodyStartControl(p *parser) {
	switch p.tok.DataAtom {
	case a.Form:
		inBodyStartForm(p)
	case a.Button:
		if p.elementInScope(defaultScope, a.Button) {
			p.generateImpliedEndTags()
			p.popUntil(defaultScope, a.Button)
		}
		p.reconstructActiveFormattingElements()
		p.addElement()
		p.framesetOK = false
	case a.Input:
		inBodyStartInput(p)
	case a.Textarea:
		p.addElement()
		p.setOriginalIM()
		p.framesetOK = false
		p.im = textIM
	case a.Select:
		if p.fragment && p.context.DataAtom == a.Select {
			// Ignore the token.
			return
		} else if p.popUntil(defaultScope, a.Select) {
			return
		}
		p.reconstructActiveFormattingElements()
		p.addElement()
		p.framesetOK = false
		return
	case a.Option:
		if p.elementInScope(defaultScope, a.Select) {
			p.generateImpliedEndTags("optgroup")
			// If oe has option element in scope, parse error?
		} else if p.top().DataAtom == a.Option {
			p.oe.pop()
		}
		p.reconstructActiveFormattingElements()
		p.addElement()
	case a.Optgroup:
		if p.elementInScope(defaultScope, a.Select) {
			p.generateImpliedEndTags()
			// If oe has option or optgroup element in scope, parse error?
		} else if p.top().DataAtom == a.Option {
			p.oe.pop()
		}
		p.reconstructActiveFormattingElements()
		p.addElement()
	}
}
