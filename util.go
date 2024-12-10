package growthbook

import (
	"net/url"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

// Checks if an experiment variation is being forced via a URL query
// string.
//
// As an example, if the id is "my-test" and url is
// http://localhost/?my-test=1, this function returns 1.
func getQueryStringOverride(id string, url *url.URL, numVariations int) (int, bool) {
	if url == nil {
		return 0, false
	}

	v, ok := url.Query()[id]
	if !ok || len(v) > 1 {
		return 0, false
	}

	vi, err := strconv.Atoi(v[0])
	if err != nil {
		return 0, false
	}

	if vi < 0 || vi >= numVariations {
		return 0, false
	}

	return vi, true
}

// This function converts slices of concrete types to []interface{}.
// This is needed to handle the common case where a user passes an
// attribute as a []string (or []int), and this needs to be compared
// against feature data deserialized from JSON, which always results
// in []interface{} slices.
func fixSliceTypes(vin interface{}) interface{} {
	// Convert all type-specific slices to interface{} slices.
	v := reflect.ValueOf(vin)
	rv := vin
	if v.Kind() == reflect.Slice || v.Kind() == reflect.Array {
		srv := make([]interface{}, v.Len())
		for i := 0; i < v.Len(); i++ {
			elem := v.Index(i).Interface()
			srv[i] = elem
		}
		rv = srv
	}
	return rv
}

type comp struct {
	actual   string
	expected string
	isPath   bool
}

func evalSimpleURLTarget(actual *url.URL, pattern string) bool {
	// If a protocol is missing, but a host is specified, add `https://`
	// to the front. Use "_____" as the wildcard since `*` is not a valid
	// hostname in some browsers
	schemeRe := regexp.MustCompile(`(?i)^([^:/?]*)\.`)
	pattern = schemeRe.ReplaceAllString(pattern, "https://$1.")
	wildcardRe := regexp.MustCompile(`\*`)
	pattern = wildcardRe.ReplaceAllLiteralString(pattern, "_____")
	expected, err := url.Parse(pattern)
	if err != nil {
		logError("Failed to parse URL pattern: ", pattern)
		return false
	}
	if expected.Host == "" {
		expected.Host = "_____"
	}

	// Compare each part of the URL separately
	comps := []comp{
		{actual.Host, expected.Host, false},
		{actual.Path, expected.Path, true},
	}
	// We only want to compare hashes if it's explicitly being targeted
	if expected.Fragment != "" {
		comps = append(comps, comp{actual.Fragment, expected.Fragment, false})
	}

	actualParams, err := url.ParseQuery(actual.RawQuery)
	if err != nil {
		logError("Failed to parse actual URL query parameters: ", actual.RawQuery)
		return false
	}
	expectedParams, err := url.ParseQuery(expected.RawQuery)
	if err != nil {
		logError("Failed to parse expected URL query parameters: ", expected.RawQuery)
		return false
	}
	for param, expectedValue := range expectedParams {
		actualValue := ""
		if actualParams.Has(param) {
			actualValue = actualParams[param][0]
		}
		comps = append(comps, comp{actualValue, expectedValue[0], false})
	}

	// If any comparisons fail, the whole thing fails
	for _, comp := range comps {
		if !evalSimpleURLPart(comp.actual, comp.expected, comp.isPath) {
			return false
		}
	}
	return true
}

func evalSimpleURLPart(actual string, pattern string, isPath bool) bool {
	// Escape special regex characters.
	specialRe := regexp.MustCompile(`([*.+?^${}()|[\]\\])`)
	escaped := specialRe.ReplaceAllString(pattern, "\\$1")
	escaped = strings.Replace(escaped, "_____", ".*", -1)

	if isPath {
		// When matching pathname, make leading/trailing slashes optional
		slashRe := regexp.MustCompile(`(^\/|\/$)`)
		escaped = slashRe.ReplaceAllLiteralString(escaped, "")
		escaped = "\\/?" + escaped + "\\/?"
	}

	escaped = "(?i)^" + escaped + "$"
	regex, err := regexp.Compile(escaped)
	if err != nil {
		logError("Failed to compile regexp: ", escaped)
		return false
	}
	return regex.MatchString(actual)
}

func getURLRegexp(regexString string) *regexp.Regexp {
	retval, err := regexp.Compile(regexString)
	if err == nil {
		return retval
	}
	logError("Failed to compile URL regexp:", err)
	return nil
}

func jsonString(v interface{}, typeName string, fieldName string) (string, bool) {
	tmp, ok := v.(string)
	if ok {
		return tmp, true
	}
	logError("Invalid JSON data type", typeName, fieldName)
	return "", false
}

func jsonBool(v interface{}, typeName string, fieldName string) (bool, bool) {
	tmp, ok := v.(bool)
	if ok {
		return tmp, true
	}
	logError("Invalid JSON data type", typeName, fieldName)
	return false, false
}

func jsonInt(v interface{}, typeName string, fieldName string) (int, bool) {
	tmp, ok := v.(float64)
	if ok {
		return int(tmp), true
	}
	logError("Invalid JSON data type", typeName, fieldName)
	return 0, false
}

func jsonFloat(v interface{}, typeName string, fieldName string) (float64, bool) {
	tmp, ok := v.(float64)
	if ok {
		return tmp, true
	}
	logError("Invalid JSON data type", typeName, fieldName)
	return 0.0, false
}

func jsonMaybeFloat(v interface{}, typeName string, fieldName string) (*float64, bool) {
	tmp, ok := v.(float64)
	if ok {
		return &tmp, true
	}
	logError("Invalid JSON data type", typeName, fieldName)
	return nil, false
}

func jsonFloatArray(v interface{}, typeName string, fieldName string) ([]float64, bool) {
	vals, ok := v.([]interface{})
	if !ok {
		logError("Invalid JSON data type", typeName, fieldName)
		return nil, false
	}
	fvals := make([]float64, len(vals))
	for i := range vals {
		tmp, ok := vals[i].(float64)
		if !ok {
			logError("Invalid JSON data type", typeName, fieldName)
			return nil, false
		}
		fvals[i] = tmp
	}
	return fvals, true
}

var (
	versionStripRe = regexp.MustCompile(`(^v|\+.*$)`)
	versionSplitRe = regexp.MustCompile(`[-.]`)
	versionNumRe   = regexp.MustCompile(`^[0-9]+$`)
)

func paddedVersionString(input string) string {
	// Remove build info and leading `v` if any
	// Split version into parts (both core version numbers and pre-release tags)
	// "v1.2.3-rc.1+build123" -> ["1","2","3","rc","1"]
	stripped := versionStripRe.ReplaceAllLiteralString(input, "")
	parts := versionSplitRe.Split(stripped, -1)

	// If it's SemVer without a pre-release, add `~` to the end
	// ["1","0","0"] -> ["1","0","0","~"]
	// "~" is the largest ASCII character, so this will make "1.0.0"
	// greater than "1.0.0-beta" for example
	if len(parts) == 3 {
		parts = append(parts, "~")
	}

	// Left pad each numeric part with spaces so string comparisons will
	// work ("9">"10", but " 9"<"10")
	for i := range parts {
		if versionNumRe.MatchString(parts[i]) {
			parts[i] = strings.Repeat(" ", 5-len(parts[i])) + parts[i]
		}
	}
	// Then, join back together into a single string
	return strings.Join(parts, "-")
}

func if0(v1 int, v2 int) int {
	if v1 == 0 {
		return v2
	}
	return v1
}

func ifnil[T int](p *T, defaultValue T) T {
	if p == nil {
		return defaultValue
	}
	return *p
}
