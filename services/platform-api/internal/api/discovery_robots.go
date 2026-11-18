package api

import (
	"bufio"
	"bytes"
	"errors"
	"net/url"
	"regexp"
	"strings"
)

// Robots rules use the longest matching directive; Allow wins equal lengths.
// A specific StageFlow group takes precedence over wildcard groups.
type robotsRule struct {
	pattern     *regexp.Regexp
	specificity int
	allow       bool
}

type discoveryRobotsGroup struct {
	agents []string
	rules  []robotsRule
}

type discoveryRobotsParser struct {
	groups    []discoveryRobotsGroup
	current   discoveryRobotsGroup
	hasRules  bool
	ruleCount int
	sitemaps  []string
}

func parseDiscoveryRobots(body []byte) ([]robotsRule, []string, error) {
	parser := discoveryRobotsParser{sitemaps: []string{}}
	scanner := bufio.NewScanner(bytes.NewReader(bytes.TrimPrefix(body, []byte("\ufeff"))))
	scanner.Buffer(make([]byte, 4096), discoveryMaxBodyBytes)

	for scanner.Scan() {
		if err := parser.readLine(scanner.Text()); err != nil {
			return nil, nil, err
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, nil, err
	}

	parser.flush()

	return selectDiscoveryRobotsRules(parser.groups), parser.sitemaps, nil
}

func (p *discoveryRobotsParser) flush() {
	if len(p.current.agents) > 0 {
		p.groups = append(p.groups, p.current)
	}

	p.current = discoveryRobotsGroup{}
	p.hasRules = false
}

func (p *discoveryRobotsParser) readLine(line string) error {
	line = strings.TrimSpace(strings.SplitN(line, "#", 2)[0])

	key, value, found := strings.Cut(line, ":")
	if !found {
		return nil
	}

	key = strings.ToLower(strings.TrimSpace(key))
	value = strings.TrimSpace(value)

	switch key {
	case "sitemap":
		if len(p.sitemaps) < discoveryMaxCandidates {
			p.sitemaps = append(p.sitemaps, value)
		}
	case "user-agent":
		if len(p.groups) >= discoveryMaxCandidates || len(p.current.agents) >= discoveryMaxCandidates {
			return errors.New("too many robots groups")
		}

		if p.hasRules {
			p.flush()
		}

		p.current.agents = append(p.current.agents, strings.ToLower(value))
	case "allow", "disallow":
		p.hasRules = true
		if len(value) > maxURLLength || p.ruleCount >= discoveryMaxCandidates {
			return errors.New("robots directives exceed limits")
		}

		if value == "" || !strings.HasPrefix(value, "/") {
			return nil
		}

		rule, err := compileDiscoveryRobotsRule(value, key == "allow")
		if err != nil {
			return err
		}

		p.ruleCount++
		p.current.rules = append(p.current.rules, rule)
	}

	return nil
}

func compileDiscoveryRobotsRule(value string, allow bool) (robotsRule, error) {
	end := strings.HasSuffix(value, "$")
	raw := normalizeRobotsOctets(strings.TrimSuffix(value, "$"))

	pieces := strings.Split(raw, "*")
	for i := range pieces {
		pieces[i] = regexp.QuoteMeta(pieces[i])
	}

	expression := "^" + strings.Join(pieces, ".*")
	if end {
		expression += "$"
	}

	pattern, err := regexp.Compile(expression)
	if err != nil {
		return robotsRule{}, err
	}

	return robotsRule{pattern: pattern, specificity: len(strings.ReplaceAll(raw, "*", "")), allow: allow}, nil
}

func selectDiscoveryRobotsRules(groups []discoveryRobotsGroup) []robotsRule {
	best := -1
	selected := []robotsRule{}

	for _, group := range groups {
		specificity := discoveryRobotsGroupSpecificity(group.agents)
		if specificity < 0 || specificity < best {
			continue
		}

		if specificity > best {
			selected = nil
			best = specificity
		}

		selected = append(selected, group.rules...)
	}

	return selected
}

func discoveryRobotsGroupSpecificity(agents []string) int {
	specificity := -1
	productToken := "stageflow-discovery"

	for _, agent := range agents {
		if agent == "*" && specificity < 0 {
			specificity = 0
		}

		if agent != "" && agent != "*" && strings.Contains(productToken, agent) && len(agent) > specificity {
			specificity = len(agent)
		}
	}

	return specificity
}

func robotsAllow(rules []robotsRule, target *url.URL) bool {
	value := target.EscapedPath()
	if value == "" {
		value = "/"
	}

	if target.RawQuery != "" {
		value += "?" + target.RawQuery
	}

	value = normalizeRobotsOctets(value)

	best := -1
	allowed := true

	for _, rule := range rules {
		if rule.pattern.MatchString(value) && rule.specificity >= best {
			if rule.specificity > best || rule.allow {
				best = rule.specificity
				allowed = rule.allow
			}
		}
	}

	return allowed
}

// Compare the same URI octets when a link percent-encodes an unreserved
// character, or a robots directive contains a literal UTF-8 character.
func normalizeRobotsOctets(value string) string {
	const hex = "0123456789ABCDEF"

	var out strings.Builder

	for i := 0; i < len(value); i++ {
		ch := value[i]
		if ch == '%' && i+2 < len(value) {
			decoded, err := url.PathUnescape(value[i : i+3])
			if err == nil {
				ch = decoded[0]
				if robotsUnreserved(ch) {
					out.WriteByte(ch)
				} else {
					out.WriteByte('%')
					out.WriteByte(hex[ch>>4])
					out.WriteByte(hex[ch&15])
				}

				i += 2

				continue
			}
		}

		if ch >= 128 {
			out.WriteByte('%')
			out.WriteByte(hex[ch>>4])
			out.WriteByte(hex[ch&15])

			continue
		}

		out.WriteByte(ch)
	}

	return out.String()
}

func robotsUnreserved(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') ||
		strings.ContainsRune("-._~", rune(ch))
}
