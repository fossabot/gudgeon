package rule

import (
	"regexp"
	"strings"

	"github.com/ryanuber/go-glob"
)

var BLOCKLIST = uint8(0)
var BLACKLIST = uint8(1)
var WHITELIST = uint8(2)

const (
	wildcard = "*"
	comment = "#"
    altComment = "//"
	regex = "/"
)

type Rule interface {
	RuleType() uint8
	IsMatch(sample string) bool
	IsComplex() bool
	Text() string
}

type baseRule struct {
	text string
	ruleType uint8
}

type textMatchRule struct {
	baseRule
}

type wildcardMatchRule struct {
	baseRule
}

type regexMatchRule struct {
	baseRule
	regexp *regexp.Regexp
}

func CreateRule(rule string, ruleType uint8) Rule {
	// a rule that starts with a comment sign is parsed as an empty string which should be ignored by other parts of the API
	if strings.HasPrefix(rule, comment) || strings.HasPrefix(rule, altComment) {
		return nil
	}

	var createdRule Rule = nil

	if strings.HasPrefix(rule, regex) && strings.HasSuffix(rule, regex) {
		// regex rules start and end with "/" to denote them that way
		createdRule = createRegexMatchRule(rule, ruleType)
	} else if strings.Contains(rule, wildcard) {
		// wildcard rules have wildcards in them (only * is supported)	
		createdRule = createWildcardMatchRule(rule, ruleType)
	} else {
		// all other rules are straight text match
		createdRule = createTextMatchRule(rule, ruleType)
	}

	// return rule
	return createdRule
}

// =================================================================
// Rule Creation
// =================================================================
func createTextMatchRule(rule string, ruleType uint8) Rule {
	newRule := new(textMatchRule)
	newRule.text = rule
	newRule.ruleType = ruleType
	return newRule
}

func createWildcardMatchRule(rule string, ruleType uint8) Rule {
	newRule := new(wildcardMatchRule)
	newRule.text = rule
	newRule.ruleType = ruleType
	return newRule
}

func createRegexMatchRule(rule string, ruleType uint8) Rule {
	newRule := new(regexMatchRule)
	newRule.text = rule
	newRule.ruleType = ruleType
	cRegex, err := regexp.Compile(rule[1:len(rule)-1])
	newRule.regexp = cRegex
	if err != nil {
		return nil
	}
	return newRule
}

// =================================================================
//
// =================================================================
func (rule *baseRule) RuleType() uint8 {
	return rule.ruleType
}

func (rule *baseRule) Text() string {
	return rule.text
}

// =================================================================
// Rule Complexity
// =================================================================
func (rule *textMatchRule) IsComplex() bool {
	return false
}

func (rule *wildcardMatchRule) IsComplex() bool {
	return true
}

func (rule *regexMatchRule) IsComplex() bool {
	return true
}

// =================================================================
// Rule Matching
// =================================================================
func (rule *textMatchRule) IsMatch(sample string) bool {
	// check to see if the value matches the rule OR if the 
	// value has a suffix that matches the "." + rule so that
	// "google.com" blocks "subdomain.google.com" and "google.com"
	return rule.text == sample || strings.HasSuffix(sample, "." + rule.text)
}

func (rule *wildcardMatchRule) IsMatch(sample string) bool {
	return glob.Glob(rule.text, sample)
}

func (rule *regexMatchRule) IsMatch(sample string) bool {
	return rule.regexp.MatchString(sample)
}