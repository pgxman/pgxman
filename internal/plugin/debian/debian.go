package debian

import (
	"strings"
)

// debNormalizedName normalizes extension name into a Debian package name which can consist of only lower case letters (a-z), digits (0-9), plus (+) and minus (-) signs, and periods (.)
// ref: https://www.debian.org/doc/debian-policy/ch-controlfields.html#:~:text=Package%20names%20(both%20source%20and,start%20with%20an%20alphanumeric%20character.
func debNormalizedName(name string) string {
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, "_", "-")

	return name
}
