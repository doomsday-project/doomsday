package storage

import (
	"regexp"
	"strings"
)

type PathList []string

//Multiple filters are "or"d together
type PathFilter struct {
	Under    []string
	Matching []string
}

func pathMatches(path, pattern string) bool {
	patternParts := strings.Split(pattern, "*")
	for i, p := range patternParts {
		patternParts[i] = regexp.QuoteMeta(p)
	}

	re := regexp.MustCompile(strings.Join(patternParts, `\A[^/:]*\Z`))
	return re.Match([]byte(path))
}

func pathIsUnder(path, dir string) bool {
	return strings.HasPrefix(strings.Trim(path, "/"), strings.TrimPrefix(dir, "/"))
}

//Doesn't modify reciever list
func (k PathList) Only(filter PathFilter) (ret PathList) {
OuterLoop:
	for _, key := range k {
		for _, match := range filter.Matching {
			if pathMatches(key, match) {
				ret = append(ret, key)
				continue OuterLoop
			}
		}

		for _, dir := range filter.Under {
			if pathIsUnder(key, dir) {
				ret = append(ret, key)
				continue OuterLoop
			}
		}
	}

	return
}

//Doesn't modify reciever list
func (k PathList) Except(filter PathFilter) (ret PathList) {
	for _, key := range k {
		var shouldNotAdd bool
		for _, match := range filter.Matching {
			if pathMatches(key, match) {
				shouldNotAdd = true
				goto DoneWithChecks
			}
		}

		for _, dir := range filter.Under {
			if pathIsUnder(key, dir) {
				shouldNotAdd = true
				goto DoneWithChecks
			}
		}

	DoneWithChecks:
		if !shouldNotAdd {
			ret = append(ret, key)
		}
	}

	return
}
