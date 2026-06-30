package agent

import "strings"

const (
	thinkOpenTag  = "[think]"
	thinkCloseTag = "[/think]"
)

// thinkingStreamParser 解析带 ... 的流式输出，区分思考与正文。
type thinkingStreamParser struct {
	state       int
	pending     strings.Builder
	contentOnly strings.Builder
}

func newThinkingStreamParser() *thinkingStreamParser {
	return &thinkingStreamParser{}
}

func (p *thinkingStreamParser) Feed(delta string, onThinking, onContent func(string) error) error {
	if delta == "" {
		return nil
	}
	p.pending.WriteString(delta)
	s := p.pending.String()
	p.pending.Reset()

	for len(s) > 0 {
		switch p.state {
		case 0:
			idx := strings.Index(s, thinkOpenTag)
			if idx >= 0 {
				s = s[idx+len(thinkOpenTag):]
				p.state = 1
				continue
			}
			keep := partialTagKeep(s, thinkOpenTag)
			if keep > 0 {
				p.pending.WriteString(s[len(s)-keep:])
				s = s[:len(s)-keep]
			}
			if s != "" {
				p.state = 2
				continue
			}
			return nil
		case 1:
			idx := strings.Index(s, thinkCloseTag)
			if idx >= 0 {
				part := s[:idx]
				if part != "" && onThinking != nil {
					if err := onThinking(part); err != nil {
						return err
					}
				}
				s = s[idx+len(thinkCloseTag):]
				p.state = 2
				continue
			}
			keep := partialTagKeep(s, thinkCloseTag)
			emit := s
			if keep > 0 {
				emit = s[:len(s)-keep]
				p.pending.WriteString(s[len(s)-keep:])
			}
			if emit != "" && onThinking != nil {
				if err := onThinking(emit); err != nil {
					return err
				}
			}
			return nil
		default:
			if onContent != nil {
				if err := onContent(s); err != nil {
					return err
				}
			}
			p.contentOnly.WriteString(s)
			return nil
		}
	}
	return nil
}

func (p *thinkingStreamParser) Flush(onThinking, onContent func(string) error) error {
	rem := p.pending.String()
	if rem == "" {
		return nil
	}
	switch p.state {
	case 1:
		if onThinking != nil {
			return onThinking(rem)
		}
	case 2:
		p.contentOnly.WriteString(rem)
		if onContent != nil {
			return onContent(rem)
		}
	default:
		p.contentOnly.WriteString(rem)
		if onContent != nil {
			return onContent(rem)
		}
	}
	return nil
}

func (p *thinkingStreamParser) Content() string {
	return strings.TrimSpace(p.contentOnly.String())
}

func partialTagKeep(s, tag string) int {
	max := len(tag) - 1
	if max <= 0 || len(s) == 0 {
		return 0
	}
	if max > len(s) {
		max = len(s)
	}
	for n := max; n > 0; n-- {
		if strings.HasPrefix(tag, s[len(s)-n:]) {
			return n
		}
	}
	return 0
}
