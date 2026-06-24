#!/bin/sh
# Integration checks for yup-sed, run inside a Debian (GNU sed) container.
#
# parity SCRIPT  — yup-sed must produce byte-identical output to GNU `sed` for
#                  the same SCRIPT applied to the same stdin.
# assert WANT    — yup-sed must produce WANT exactly (used where yup-sed diverges
#                  from GNU by design; see cmd-sed COMPATIBILITY.md).
#
# yup-sed implements only the GNU `sed` s/// substitution command with the
# g, i, p, and N (Nth-match) flags. Crucially, its regular-expression engine
# is Go's RE2 (an ERE-like flavor with $-based replacement refs), NOT GNU's
# BRE. So plain, BRE/ERE-agnostic substitutions reach parity, while anything
# touching BRE syntax, GNU `\1`/`&` replacements, or non-s commands diverges.
# LC_ALL=C pins byte semantics on both sides.
set -eu
export LC_ALL=C

fails=0

# parity SCRIPT INPUT: compare yup-sed and GNU sed on the same stdin.
parity() {
	script=$1
	input=$2
	ours=$(printf '%s' "$input" | yup-sed "$script" 2>/dev/null || true)
	gnu=$(printf '%s' "$input" | sed "$script" 2>/dev/null || true)
	if [ "$ours" = "$gnu" ]; then
		printf 'ok    parity  sed %s\n' "$script"
	else
		printf 'FAIL  parity  sed %s\n        gnu:  %s\n        ours: %s\n' "$script" "$gnu" "$ours"
		fails=$((fails + 1))
	fi
}

# assert WANT SCRIPT INPUT: yup-sed must emit WANT exactly (documented divergence).
assert() {
	want=$1
	script=$2
	input=$3
	got=$(printf '%s' "$input" | yup-sed "$script" 2>/dev/null || true)
	if [ "$got" = "$want" ]; then
		printf 'ok    assert  sed %s\n' "$script"
	else
		printf 'FAIL  assert  sed %s\n        want: %s\n        got:  %s\n' "$script" "$want" "$got"
		fails=$((fails + 1))
	fi
}

# --- Parity: substitutions whose syntax is shared by RE2 and GNU BRE/ERE. ---

# Basic first-occurrence substitution.
parity 's/a/x/' 'aaa
'
# Global flag g: every match on the line.
parity 's/a/x/g' 'banana
'
# Ignore-case flag i.
parity 's/abc/x/i' 'ABC
'
# Nth-match flag N: replace only the 2nd match.
parity 's/a/x/2' 'aaa
'
# Print flag p: emit the line again when a substitution fired.
parity 's/b/X/p' 'abc
'
# Alternate delimiter (any byte may delimit).
parity 's|l|L|g' 'hello
'
# Metacharacter . (any char) — same meaning in RE2 and BRE.
parity 's/a.b/X/' 'a.b
'
# Anchors ^ and $.
parity 's/^/> /' 'cat
'
parity 's/$/!/' 'cat
'
# Bracket class [0-9] (shared) replacing a literal space.
parity 's/ /_/g' 'one two
'
# Replacement containing & when the pattern does NOT match: no-op on both.
parity 's/zzz/Q/' 'a&b
'

# --- Documented divergences (see cmd-sed COMPATIBILITY.md). ---

# RE2 is ERE-like: + is the one-or-more operator, so [0-9]+ matches digits.
# GNU sed's default BRE treats + literally, so it would NOT match here.
assert 'fooN' 's/[0-9]+/N/' 'foo123
'
# RE2: unescaped ( ) form a capture group; GNU BRE needs \( \) and treats
# ( ) as literals. So ours captures and rewrites where GNU would not.
assert 'X' 's/(a)b/X/' 'ab
'
# Replacement back-reference syntax is Go's $1 / ${1}, not GNU's \1.
assert 'a' 's/(a)b/$1/' 'ab
'
assert 'ba' 's/(a)(b)/${2}${1}/' 'ab
'
# GNU \1 in the replacement is taken literally by RE2 (no such expansion).
assert '[\1][\1]' 's/(a)b/[\1]/g' 'abab
'
# & in the replacement is literal in RE2; GNU expands it to the whole match.
assert '[&]at' 's/c/[&]/' 'cat
'
# Non-s commands are unsupported: the wrapper fails the stream with its sentinel.
assert '' 'd' 'x
'

if [ "$fails" -ne 0 ]; then
	printf '\n%s check(s) failed\n' "$fails"
	exit 1
fi
printf '\nall checks passed\n'
