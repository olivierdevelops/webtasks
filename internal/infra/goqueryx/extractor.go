// Package goqueryx evaluates extract specs against HTML using goquery (a Go
// port of jQuery selectors). Mirrors the Java JsoupExtractor.
package goqueryx

import (
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"webtasks/internal/domain"
)

type Extractor struct{}

// ExtractObject returns one record built from `spec.Fields` applied against
// the first match of `spec.Selector` (or the document body if `.`).
func (e Extractor) ExtractObject(html string, spec domain.ExtractSpec) (map[string]any, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, err
	}
	root := doc.Selection
	if spec.Selector != "" && spec.Selector != "." {
		root = doc.Find(spec.Selector).First()
	}
	if root.Length() == 0 {
		return map[string]any{}, nil
	}
	return e.extractFrom(root, spec.Fields), nil
}

// ExtractList returns one record per match of `spec.Selector`.
func (e Extractor) ExtractList(html string, spec domain.ExtractSpec) ([]map[string]any, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0)
	doc.Find(spec.Selector).Each(func(_ int, s *goquery.Selection) {
		out = append(out, e.extractFrom(s, spec.Fields))
	})
	return out, nil
}

func (e Extractor) extractFrom(ctx *goquery.Selection, fields map[string]domain.ExtractFieldSpec) map[string]any {
	obj := make(map[string]any, len(fields))
	for name, fs := range fields {
		obj[name] = e.extractField(ctx, fs)
	}
	return obj
}

func (e Extractor) extractField(ctx *goquery.Selection, fs domain.ExtractFieldSpec) any {
	var target *goquery.Selection
	if fs.Selector == "" || fs.Selector == "." {
		target = ctx
	} else {
		target = ctx.Find(fs.Selector).First()
	}

	var raw any
	switch fs.Kind {
	case "const":
		raw = fs.ConstValue
	case "attr":
		if target.Length() == 0 {
			return nil
		}
		v, ok := target.Attr(fs.AttrName)
		if !ok {
			return nil
		}
		raw = v
	case "html":
		if target.Length() == 0 {
			return nil
		}
		h, _ := target.Html()
		raw = h
	default: // "text"
		if target.Length() == 0 {
			return nil
		}
		raw = strings.TrimSpace(target.Text())
	}
	if fs.Transform == "" {
		return raw
	}
	return applyTransform(raw, fs.Transform)
}

func applyTransform(value any, transform string) any {
	if value == nil {
		return nil
	}
	s, _ := value.(string)
	switch transform {
	case "int":
		v, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
		if err != nil {
			return nil
		}
		return v
	case "long":
		v, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
		if err != nil {
			return nil
		}
		return v
	case "trim":
		return strings.TrimSpace(s)
	case "lower":
		return strings.ToLower(s)
	case "upper":
		return strings.ToUpper(s)
	}
	return value
}
