//-----------------------------------------------------------------------------
// Copyright (c) 2020 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package zettelmark_test provides some tests for the zettelmarkup parser.
package zettelmark_test

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"zettelstore.de/z/ast"
	"zettelstore.de/z/input"
	"zettelstore.de/z/parser"
)

type TestCase struct{ source, want string }
type TestCases []TestCase

func replace(s string, tcs TestCases) TestCases {
	var testCases TestCases

	for _, tc := range tcs {
		source := strings.ReplaceAll(tc.source, "$", s)
		want := strings.ReplaceAll(tc.want, "$", s)
		testCases = append(testCases, TestCase{source, want})
	}
	return testCases
}

func checkTcs(t *testing.T, tcs TestCases) {
	t.Helper()

	for tcn, tc := range tcs {
		t.Run(fmt.Sprintf("TC=%02d,src=%q", tcn, tc.source), func(st *testing.T) {
			st.Helper()
			inp := input.NewInput(tc.source)
			bns := parser.ParseBlocks(inp, nil, "zmk")
			var tv TestVisitor
			tv.visitBlockSlice(bns)
			got := tv.String()
			if tc.want != got {
				st.Errorf("\nwant=%q\n got=%q", tc.want, got)
			}
		})
	}
}

func TestEOL(t *testing.T) {
	checkTcs(t, TestCases{
		{"", ""},
		{"\n", ""},
		{"\r", ""},
		{"\r\n", ""},
		{"\n\n", ""},
	})
}

func TestText(t *testing.T) {
	checkTcs(t, TestCases{
		{"abcd", "(PARA abcd)"},
		{"ab cd", "(PARA ab SP cd)"},
		{"abcd ", "(PARA abcd)"},
		{" abcd", "(PARA abcd)"},
		{"\\", "(PARA \\)"},
		{"\\\n", ""},
		{"\\\ndef", "(PARA HB def)"},
		{"\\\r", ""},
		{"\\\rdef", "(PARA HB def)"},
		{"\\\r\n", ""},
		{"\\\r\ndef", "(PARA HB def)"},
		{"\\a", "(PARA a)"},
		{"\\aa", "(PARA aa)"},
		{"a\\a", "(PARA aa)"},
		{"\\+", "(PARA +)"},
		{"\\ ", "(PARA \u00a0)"},
		{"...", "(PARA \u2026)"},
		{"...,", "(PARA \u2026,)"},
		{"...;", "(PARA \u2026;)"},
		{"...:", "(PARA \u2026:)"},
		{"...!", "(PARA \u2026!)"},
		{"...?", "(PARA \u2026?)"},
		{"...-", "(PARA ...-)"},
		{"a...b", "(PARA a...b)"},
	})
}

func TestSpace(t *testing.T) {
	checkTcs(t, TestCases{
		{" ", ""},
		{"\t", ""},
		{"  ", ""},
	})
}

func TestSoftBreak(t *testing.T) {
	checkTcs(t, TestCases{
		{"x\ny", "(PARA x SB y)"},
		{"z\n", "(PARA z)"},
		{" \n ", ""},
		{" \n", ""},
	})
}

func TestHardBreak(t *testing.T) {
	checkTcs(t, TestCases{
		{"x  \ny", "(PARA x HB y)"},
		{"z  \n", "(PARA z)"},
		{"   \n ", ""},
		{"   \n", ""},
	})
}

func TestLink(t *testing.T) {
	checkTcs(t, TestCases{
		{"[", "(PARA [)"},
		{"[[", "(PARA [[)"},
		{"[[|", "(PARA [[|)"},
		{"[[]", "(PARA [[])"},
		{"[[|]", "(PARA [[|])"},
		{"[[]]", "(PARA [[]])"},
		{"[[|]]", "(PARA [[|]])"},
		{"[[ ]]", "(PARA [[ SP ]])"},
		{"[[\n]]", "(PARA [[ SB ]])"},
		{"[[ a]]", "(PARA (LINK a a))"},
		{"[[a ]]", "(PARA [[a SP ]])"},
		{"[[a\n]]", "(PARA [[a SB ]])"},
		{"[[a]]", "(PARA (LINK a a))"},
		{"[[12345678901234]]", "(PARA (LINK 12345678901234 12345678901234))"},
		{"[[a]", "(PARA [[a])"},
		{"[[|a]]", "(PARA [[|a]])"},
		{"[[b|]]", "(PARA [[b|]])"},
		{"[[b|a]]", "(PARA (LINK a b))"},
		{"[[b| a]]", "(PARA (LINK a b))"},
		{"[[b%c|a]]", "(PARA (LINK a b%c))"},
		{"[[b%%c|a]]", "(PARA [[b {% c|a]]})"},
		{"[[b|a]", "(PARA [[b|a])"},
		{"[[b\nc|a]]", "(PARA (LINK a b SB c))"},
		{"[[b c|a#n]]", "(PARA (LINK a#n b SP c))"},
		{"[[a]]go", "(PARA (LINK a a) go)"},
		{"[[a]]{go}", "(PARA (LINK a a)[ATTR go])"},
		{"[[[[a]]|b]]", "(PARA (LINK [[a [[a) |b]])"},
	})
}

func TestCite(t *testing.T) {
	checkTcs(t, TestCases{
		{"[@", "(PARA [@)"},
		{"[@]", "(PARA [@])"},
		{"[@a]", "(PARA (CITE a))"},
		{"[@ a]", "(PARA [@ SP a])"},
		{"[@a ]", "(PARA (CITE a))"},
		{"[@a\n]", "(PARA (CITE a))"},
		{"[@a\nx]", "(PARA (CITE a SB x))"},
		{"[@a\n\n]", "(PARA [@a)(PARA ])"},
		{"[@a,\n]", "(PARA (CITE a))"},
		{"[@a,n]", "(PARA (CITE a n))"},
		{"[@a| n]", "(PARA (CITE a n))"},
		{"[@a|n ]", "(PARA (CITE a n))"},
		{"[@a,[@b]]", "(PARA (CITE a (CITE b)))"},
		{"[@a]{color=green}", "(PARA (CITE a)[ATTR color=green])"},
	})
}

func TestFootnote(t *testing.T) {
	checkTcs(t, TestCases{
		{"[^", "(PARA [^)"},
		{"[^]", "(PARA (FN))"},
		{"[^abc]", "(PARA (FN abc))"},
		{"[^abc ]", "(PARA (FN abc))"},
		{"[^abc\ndef]", "(PARA (FN abc SB def))"},
		{"[^abc\n\ndef]", "(PARA [^abc)(PARA def])"},
		{"[^abc[^def]]", "(PARA (FN abc (FN def)))"},
		{"[^abc]{-}", "(PARA (FN abc)[ATTR -])"},
	})
}

func TestImage(t *testing.T) {
	checkTcs(t, TestCases{
		{"{", "(PARA {)"},
		{"{{", "(PARA {{)"},
		{"{{|", "(PARA {{|)"},
		{"{{}", "(PARA {{})"},
		{"{{|}", "(PARA {{|})"},
		{"{{}}", "(PARA {{}})"},
		{"{{|}}", "(PARA {{|}})"},
		{"{{ }}", "(PARA {{ SP }})"},
		{"{{\n}}", "(PARA {{ SB }})"},
		{"{{a }}", "(PARA {{a SP }})"},
		{"{{a\n}}", "(PARA {{a SB }})"},
		{"{{a}}", "(PARA (IMAGE a))"},
		{"{{12345678901234}}", "(PARA (IMAGE 12345678901234))"},
		{"{{ a}}", "(PARA (IMAGE a))"},
		{"{{a}", "(PARA {{a})"},
		{"{{|a}}", "(PARA {{|a}})"},
		{"{{b|}}", "(PARA {{b|}})"},
		{"{{b|a}}", "(PARA (IMAGE a b))"},
		{"{{b| a}}", "(PARA (IMAGE a b))"},
		{"{{b|a}", "(PARA {{b|a})"},
		{"{{b\nc|a}}", "(PARA (IMAGE a b SB c))"},
		{"{{b c|a#n}}", "(PARA (IMAGE a#n b SP c))"},
		{"{{a}}{go}", "(PARA (IMAGE a)[ATTR go])"},
		{"{{{{a}}|b}}", "(PARA (IMAGE %7B%7Ba) |b}})"},
	})
}

func TestTag(t *testing.T) {
	checkTcs(t, TestCases{
		{"#", "(PARA #)"},
		{"##", "(PARA ##)"},
		{"###", "(PARA ###)"},
		{"#tag", "(PARA #tag#)"},
		{"#tag,", "(PARA #tag# ,)"},
		{"#t-g ", "(PARA #t-g#)"},
		{"#t_g", "(PARA #t_g#)"},
	})
}

func TestMark(t *testing.T) {
	checkTcs(t, TestCases{
		{"[!", "(PARA [!)"},
		{"[!\n", "(PARA [!)"},
		{"[!]", "(PARA (MARK *))"},
		{"[! ]", "(PARA [! SP ])"},
		{"[!a]", "(PARA (MARK a))"},
		{"[!a ]", "(PARA [!a SP ])"},
		{"[!a_]", "(PARA (MARK a_))"},
		{"[!a-b]", "(PARA (MARK a-b))"},
		{"[!a][!a]", "(PARA (MARK a) (MARK a-1))"},
		{"[!][!]", "(PARA (MARK *) (MARK *-1))"},
	})
}

func TestComment(t *testing.T) {
	checkTcs(t, TestCases{
		{"%", "(PARA %)"},
		{"%%", "(PARA {%})"},
		{"%\n", "(PARA %)"},
		{"%%\n", "(PARA {%})"},
		{"%%a", "(PARA {% a})"},
		{"%%%a", "(PARA {% a})"},
		{"%% a", "(PARA {% a})"},
		{"%%%  a", "(PARA {% a})"},
		{"%% % a", "(PARA {% % a})"},
		{"%%a", "(PARA {% a})"},
		{"a%%b", "(PARA a {% b})"},
		{"a %%b", "(PARA a SP {% b})"},
		{" %%b", "(PARA {% b})"},
		{"%%b ", "(PARA {% b })"},
		{"100%", "(PARA 100%)"},
	})
}

func TestFormat(t *testing.T) {
	for _, ch := range []string{"/", "*", "_", "~", "'", "^", ",", "<", "\"", ";", ":"} {
		checkTcs(t, replace(ch, TestCases{
			{"$", "(PARA $)"},
			{"$$", "(PARA $$)"},
			{"$$$", "(PARA $$$)"},
			{"$$$$", "(PARA {$})"},
			{"$$a$$", "(PARA {$ a})"},
			{"$$a$$$", "(PARA {$ a} $)"},
			{"$$$a$$", "(PARA {$ $a})"},
			{"$$$a$$$", "(PARA {$ $a} $)"},
			{"$\\$", "(PARA $$)"},
			{"$\\$$", "(PARA $$$)"},
			{"$$\\$", "(PARA $$$)"},
			{"$$a\\$$", "(PARA $$a$$)"},
			{"$$a$\\$", "(PARA $$a$$)"},
			{"$$a\\$$$", "(PARA {$ a$})"},
			{"$$a\na$$", "(PARA {$ a SB a})"},
			{"$$a\n\na$$", "(PARA $$a)(PARA a$$)"},
			{"$$a$${go}", "(PARA {$ a}[ATTR go])"},
		}))
	}
	checkTcs(t, TestCases{
		{"//****//", "(PARA {/ {*}})"},
		{"//**a**//", "(PARA {/ {* a}})"},
		{"//**//**", "(PARA // {* //})"},
	})
}

func TestLiteral(t *testing.T) {
	for _, ch := range []string{"`", "+", "="} {
		checkTcs(t, replace(ch, TestCases{
			{"$", "(PARA $)"},
			{"$$", "(PARA $$)"},
			{"$$$", "(PARA $$$)"},
			{"$$$$", "(PARA {$})"},
			{"$$a$$", "(PARA {$ a})"},
			{"$$a$$$", "(PARA {$ a} $)"},
			{"$$$a$$", "(PARA {$ $a})"},
			{"$$$a$$$", "(PARA {$ $a} $)"},
			{"$\\$", "(PARA $$)"},
			{"$\\$$", "(PARA $$$)"},
			{"$$\\$", "(PARA $$$)"},
			{"$$a\\$$", "(PARA $$a$$)"},
			{"$$a$\\$", "(PARA $$a$$)"},
			{"$$a\\$$$", "(PARA {$ a$})"},
			{"$$a$${go}", "(PARA {$ a}[ATTR go])"},
		}))
	}
	checkTcs(t, TestCases{
		{"++````++", "(PARA {+ ````})"},
		{"++``a``++", "(PARA {+ ``a``})"},
		{"++``++``", "(PARA {+ ``} ``)"},
		{"++\\+++", "(PARA {+ +})"},
	})
}

func TestMixFormatCode(t *testing.T) {
	checkTcs(t, TestCases{
		{"//abc//\n**def**", "(PARA {/ abc} SB {* def})"},
		{"++abc++\n==def==", "(PARA {+ abc} SB {= def})"},
		{"//abc//\n==def==", "(PARA {/ abc} SB {= def})"},
		{"//abc//\n``def``", "(PARA {/ abc} SB {` def})"},
		{"\"\"ghi\"\"\n::abc::\n``def``\n", "(PARA {\" ghi} SB {: abc} SB {` def})"},
	})
}

func TestNDash(t *testing.T) {
	checkTcs(t, TestCases{
		{"--", "(PARA \u2013)"},
		{"a--b", "(PARA a\u2013b)"},
	})
}

func TestEntity(t *testing.T) {
	checkTcs(t, TestCases{
		{"&", "(PARA &)"},
		{"&;", "(PARA &;)"},
		{"&#;", "(PARA &#;)"},
		{"&#1a;", "(PARA & #1a# ;)"},
		{"&#x;", "(PARA & #x# ;)"},
		{"&#x0z;", "(PARA & #x0z# ;)"},
		{"&1;", "(PARA &1;)"},
		// Good cases
		{"&lt;", "(PARA <)"},
		{"&#48;", "(PARA 0)"},
		{"&#x4A;", "(PARA J)"},
		{"&#X4a;", "(PARA J)"},
		{"&hellip;", "(PARA \u2026)"},
		{"E: &amp;,&#13;;&#xa;.", "(PARA E: SP &,\r;\n.)"},
	})
}

func TestVerbatim(t *testing.T) {
	checkTcs(t, TestCases{
		{"```\n```", "(PROG)"},
		{"```\nabc\n```", "(PROG\nabc)"},
		{"```\nabc\n````", "(PROG\nabc)"},
		{"````\nabc\n````", "(PROG\nabc)"},
		{"````\nabc\n```\n````", "(PROG\nabc\n```)"},
		{"````go\nabc\n````", "(PROG\nabc)[ATTR =go]"},
	})
}

func TestSpanRegion(t *testing.T) {
	checkTcs(t, TestCases{
		{":::\n:::", "(SPAN)"},
		{":::\nabc\n:::", "(SPAN (PARA abc))"},
		{":::\nabc\n::::", "(SPAN (PARA abc))"},
		{"::::\nabc\n::::", "(SPAN (PARA abc))"},
		{"::::\nabc\n:::\ndef\n:::\n::::", "(SPAN (PARA abc)(SPAN (PARA def)))"},
		{":::{go}\n:::", "(SPAN)[ATTR go]"},
		{":::\nabc\n::: def ", "(SPAN (PARA abc) (LINE def))"},
	})
}

func TestQuoteRegion(t *testing.T) {
	checkTcs(t, TestCases{
		{"<<<\n<<<", "(QUOTE)"},
		{"<<<\nabc\n<<<", "(QUOTE (PARA abc))"},
		{"<<<\nabc\n<<<<", "(QUOTE (PARA abc))"},
		{"<<<<\nabc\n<<<<", "(QUOTE (PARA abc))"},
		{"<<<<\nabc\n<<<\ndef\n<<<\n<<<<", "(QUOTE (PARA abc)(QUOTE (PARA def)))"},
		{"<<<go\n<<<", "(QUOTE)[ATTR =go]"},
		{"<<<\nabc\n<<< def ", "(QUOTE (PARA abc) (LINE def))"},
	})
}

func TestVerseRegion(t *testing.T) {
	checkTcs(t, replace("\"", TestCases{
		{"$$$\n$$$", "(VERSE)"},
		{"$$$\nabc\n$$$", "(VERSE (PARA abc))"},
		{"$$$\nabc\n$$$$", "(VERSE (PARA abc))"},
		{"$$$$\nabc\n$$$$", "(VERSE (PARA abc))"},
		{"$$$\nabc\ndef\n$$$", "(VERSE (PARA abc HB def))"},
		{"$$$$\nabc\n$$$\ndef\n$$$\n$$$$", "(VERSE (PARA abc)(VERSE (PARA def)))"},
		{"$$$go\n$$$", "(VERSE)[ATTR =go]"},
		{"$$$\nabc\n$$$ def ", "(VERSE (PARA abc) (LINE def))"},
	}))
}

func TestHeading(t *testing.T) {
	checkTcs(t, TestCases{
		{"=h", "(PARA =h)"},
		{"= h", "(PARA = SP h)"},
		{"==h", "(PARA ==h)"},
		{"== h", "(PARA == SP h)"},
		{"===h", "(PARA ===h)"},
		{"=== h", "(H2 h)"},
		{"===  h", "(H2 h)"},
		{"==== h", "(H3 h)"},
		{"===== h", "(H4 h)"},
		{"====== h", "(H5 h)"},
		{"======= h", "(H6 h)"},
		{"======== h", "(H6 h)"},
		{"=", "(PARA =)"},
		{"=== h=//=a//", "(H2 h= {/ =a})"},
		{"=\n", "(PARA =)"},
		{"a=", "(PARA a=)"},
		{" =", "(PARA =)"},
		{"=== h\na", "(H2 h)(PARA a)"},
		{"=== h i {-}", "(H2 h SP i)[ATTR -]"},
	})
}

func TestHRule(t *testing.T) {
	checkTcs(t, TestCases{
		{"-", "(PARA -)"},
		{"---", "(HR)"},
		{"----", "(HR)"},
		{"---A", "(HR)[ATTR =A]"},
		{"---A-", "(HR)[ATTR =A-]"},
		{"-1", "(PARA -1)"},
		{"2-1", "(PARA 2-1)"},
		{"---  {  go  }  ", "(HR)[ATTR go]"},
		{"---  {  .go  }  ", "(HR)[ATTR class=go]"},
	})
}

func TestList(t *testing.T) {
	// No ">" in the following, because quotation lists may have empty items.
	for _, ch := range []string{"*", "#"} {
		checkTcs(t, replace(ch, TestCases{
			{"$", "(PARA $)"},
			{"$$", "(PARA $$)"},
			{"$$$", "(PARA $$$)"},
			{"$ ", "(PARA $)"},
			{"$$ ", "(PARA $$)"},
			{"$$$ ", "(PARA $$$)"},
		}))
	}
	checkTcs(t, TestCases{
		{"* abc", "(UL {(PARA abc)})"},
		{"** abc", "(UL {(UL {(PARA abc)})})"},
		{"*** abc", "(UL {(UL {(UL {(PARA abc)})})})"},
		{"**** abc", "(UL {(UL {(UL {(UL {(PARA abc)})})})})"},
		{"** abc\n**** def", "(UL {(UL {(PARA abc)(UL {(UL {(PARA def)})})})})"},
		{"* abc\ndef", "(UL {(PARA abc)})(PARA def)"},
		{"* abc\n def", "(UL {(PARA abc)})(PARA def)"},
		{"* abc\n* def", "(UL {(PARA abc)} {(PARA def)})"},
		{"* abc\n  def", "(UL {(PARA abc SB def)})"},
		{"* abc\n   def", "(UL {(PARA abc SB def)})"},
		{"* abc\n\ndef", "(UL {(PARA abc)})(PARA def)"},
		{"* abc\n\n def", "(UL {(PARA abc)})(PARA def)"},
		{"* abc\n\n  def", "(UL {(PARA abc)(PARA def)})"},
		{"* abc\n\n   def", "(UL {(PARA abc)(PARA def)})"},
		{"* abc\n** def", "(UL {(PARA abc)(UL {(PARA def)})})"},
		{"* abc\n** def\n* ghi", "(UL {(PARA abc)(UL {(PARA def)})} {(PARA ghi)})"},
		{"* abc\n\n  def\n* ghi", "(UL {(PARA abc)(PARA def)} {(PARA ghi)})"},
		{"* abc\n** def\n   ghi\n  jkl", "(UL {(PARA abc)(UL {(PARA def SB ghi)})(PARA jkl)})"},

		// A list does not last beyond a region
		{":::\n# abc\n:::\n# def", "(SPAN (OL {(PARA abc)}))(OL {(PARA def)})"},

		// A HRule creates a new list
		{"* abc\n---\n* def", "(UL {(PARA abc)})(HR)(UL {(PARA def)})"},

		// Changing list type adds a new list
		{"* abc\n# def", "(UL {(PARA abc)})(OL {(PARA def)})"},

		// Quotation lists mayx have empty items
		{">", "(QL {})"},
	})
}

func TestEnumAfterPara(t *testing.T) {
	checkTcs(t, TestCases{
		{"abc\n* def", "(PARA abc)(UL {(PARA def)})"},
		{"abc\n*def", "(PARA abc SB *def)"},
	})
}

func TestDefinition(t *testing.T) {
	checkTcs(t, TestCases{
		{";", "(PARA ;)"},
		{"; ", "(PARA ;)"},
		{"; abc", "(DL (DT abc))"},
		{"; abc\ndef", "(DL (DT abc))(PARA def)"},
		{"; abc\n def", "(DL (DT abc))(PARA def)"},
		{"; abc\n  def", "(DL (DT abc SB def))"},
		{":", "(PARA :)"},
		{": ", "(PARA :)"},
		{": abc", "(PARA : SP abc)"},
		{"; abc\n: def", "(DL (DT abc) (DD (PARA def)))"},
		{"; abc\n: def\nghi", "(DL (DT abc) (DD (PARA def)))(PARA ghi)"},
		{"; abc\n: def\n ghi", "(DL (DT abc) (DD (PARA def)))(PARA ghi)"},
		{"; abc\n: def\n  ghi", "(DL (DT abc) (DD (PARA def SB ghi)))"},
		{"; abc\n: def\n\n  ghi", "(DL (DT abc) (DD (PARA def)(PARA ghi)))"},
		{"; abc\n:", "(DL (DT abc))(PARA :)"},
		{"; abc\n: def\n: ghi", "(DL (DT abc) (DD (PARA def)) (DD (PARA ghi)))"},
		{"; abc\n: def\n; ghi\n: jkl", "(DL (DT abc) (DD (PARA def)) (DT ghi) (DD (PARA jkl)))"},
	})
}

func TestTable(t *testing.T) {
	checkTcs(t, TestCases{
		{"|", "(TAB (TR))"},
		{"|a", "(TAB (TR (TD a)))"},
		{"|a|", "(TAB (TR (TD a)))"},
		{"|a| ", "(TAB (TR (TD a)(TD)))"},
		{"|a|b", "(TAB (TR (TD a)(TD b)))"},
		{"|a|b\n|c|d", "(TAB (TR (TD a)(TD b))(TR (TD c)(TD d)))"},
	})
}

func TestBlockAttr(t *testing.T) {
	checkTcs(t, TestCases{
		{":::go\n:::", "(SPAN)[ATTR =go]"},
		{":::go=\n:::", "(SPAN)[ATTR =go]"},
		{":::{}\n:::", "(SPAN)"},
		{":::{ }\n:::", "(SPAN)"},
		{":::{.go}\n:::", "(SPAN)[ATTR class=go]"},
		{":::{=go}\n:::", "(SPAN)[ATTR =go]"},
		{":::{go}\n:::", "(SPAN)[ATTR go]"},
		{":::{go=py}\n:::", "(SPAN)[ATTR go=py]"},
		{":::{.go=py}\n:::", "(SPAN)"},
		{":::{go=}\n:::", "(SPAN)[ATTR go]"},
		{":::{.go=}\n:::", "(SPAN)"},
		{":::{go py}\n:::", "(SPAN)[ATTR go py]"},
		{":::{go\npy}\n:::", "(SPAN (PARA py}))"},
		{":::{.go py}\n:::", "(SPAN)[ATTR class=go py]"},
		{":::{go .py}\n:::", "(SPAN)[ATTR class=py go]"},
		{":::{.go py=3}\n:::", "(SPAN)[ATTR class=go py=3]"},
		{":::  {  go  }  \n:::", "(SPAN)[ATTR go]"},
		{":::  {  .go  }  \n:::", "(SPAN)[ATTR class=go]"},
	})
	checkTcs(t, replace("\"", TestCases{
		{":::{py=3}\n:::", "(SPAN)[ATTR py=3]"},
		{":::{py=$2 3$}\n:::", "(SPAN)[ATTR py=$2 3$]"},
		{":::{py=$2\\$3$}\n:::", "(SPAN)[ATTR py=2$3]"},
		{":::{py=2$3}\n:::", "(SPAN)[ATTR py=2$3]"},
		{":::{py=$2\n3$}\n:::", "(SPAN (PARA 3$}))"},
		{":::{py=$2 3}\n:::", "(SPAN)"},
		{":::{py=2 py=3}\n:::", "(SPAN)[ATTR py=$2 3$]"},
		{":::{.go .py}\n:::", "(SPAN)[ATTR class=$go py$]"},
		{":::{go go}\n:::", "(SPAN)[ATTR go]"},
		{":::{=py =go}\n:::", "(SPAN)[ATTR =go]"},
	}))
}

func TestInlineAttr(t *testing.T) {
	checkTcs(t, TestCases{
		{"::a::{}", "(PARA {: a})"},
		{"::a::{ }", "(PARA {: a})"},
		{"::a::{.go}", "(PARA {: a}[ATTR class=go])"},
		{"::a::{=go}", "(PARA {: a}[ATTR =go])"},
		{"::a::{go}", "(PARA {: a}[ATTR go])"},
		{"::a::{go=py}", "(PARA {: a}[ATTR go=py])"},
		{"::a::{.go=py}", "(PARA {: a} {.go=py})"},
		{"::a::{go=}", "(PARA {: a}[ATTR go])"},
		{"::a::{.go=}", "(PARA {: a} {.go=})"},
		{"::a::{go py}", "(PARA {: a}[ATTR go py])"},
		{"::a::{go\npy}", "(PARA {: a}[ATTR go py])"},
		{"::a::{.go py}", "(PARA {: a}[ATTR class=go py])"},
		{"::a::{go .py}", "(PARA {: a}[ATTR class=py go])"},
		{"::a::{  \n go \n .py\n  \n}", "(PARA {: a}[ATTR class=py go])"},
		{"::a::{  \n go \n .py\n\n}", "(PARA {: a}[ATTR class=py go])"},
		{"::a::{\ngo\n}", "(PARA {: a}[ATTR go])"},
	})
	checkTcs(t, replace("\"", TestCases{
		{"::a::{py=3}", "(PARA {: a}[ATTR py=3])"},
		{"::a::{py=$2 3$}", "(PARA {: a}[ATTR py=$2 3$])"},
		{"::a::{py=$2\\$3$}", "(PARA {: a}[ATTR py=2$3])"},
		{"::a::{py=2$3}", "(PARA {: a}[ATTR py=2$3])"},
		{"::a::{py=$2\n3$}", "(PARA {: a}[ATTR py=$2 3$])"},
		{"::a::{py=$2 3}", "(PARA {: a} {py=$2 SP 3})"},

		{"::a::{py=2 py=3}", "(PARA {: a}[ATTR py=$2 3$])"},
		{"::a::{.go .py}", "(PARA {: a}[ATTR class=$go py$])"},
	}))
}

func TestTemp(t *testing.T) {
	checkTcs(t, TestCases{
		{"", ""},
	})
}

// --------------------------------------------------------------------------

// TestVisitor serializes the abstract syntax tree to a string.
type TestVisitor struct {
	b strings.Builder
}

func (tv *TestVisitor) String() string { return tv.b.String() }
func (tv *TestVisitor) VisitPara(pn *ast.ParaNode) {
	tv.b.WriteString("(PARA")
	tv.visitInlineSlice(pn.Inlines)
	tv.b.WriteByte(')')
}

var mapVerbatimCode = map[ast.VerbatimCode]string{
	ast.VerbatimProg: "(PROG",
}

func (tv *TestVisitor) VisitVerbatim(vn *ast.VerbatimNode) {
	code, ok := mapVerbatimCode[vn.Code]
	if !ok {
		panic(fmt.Sprintf("Unknown verbatim code %v", vn.Code))
	}
	tv.b.WriteString(code)
	for _, line := range vn.Lines {
		tv.b.WriteByte('\n')
		tv.b.WriteString(line)
	}
	tv.b.WriteByte(')')
	tv.visitAttributes(vn.Attrs)
}

var mapRegionCode = map[ast.RegionCode]string{
	ast.RegionSpan:  "(SPAN",
	ast.RegionQuote: "(QUOTE",
	ast.RegionVerse: "(VERSE",
}

// VisitRegion stores information about a region.
func (tv *TestVisitor) VisitRegion(rn *ast.RegionNode) {
	code, ok := mapRegionCode[rn.Code]
	if !ok {
		panic(fmt.Sprintf("Unknown region code %v", rn.Code))
	}
	tv.b.WriteString(code)
	if rn.Blocks != nil {
		tv.b.WriteByte(' ')
		tv.visitBlockSlice(rn.Blocks)
	}
	if len(rn.Inlines) > 0 {
		tv.b.WriteString(" (LINE")
		tv.visitInlineSlice(rn.Inlines)
		tv.b.WriteByte(')')
	}
	tv.b.WriteByte(')')
	tv.visitAttributes(rn.Attrs)
}

func (tv *TestVisitor) VisitHeading(hn *ast.HeadingNode) {
	fmt.Fprintf(&tv.b, "(H%d", hn.Level)
	tv.visitInlineSlice(hn.Inlines)
	tv.b.WriteByte(')')
	tv.visitAttributes(hn.Attrs)
}
func (tv *TestVisitor) VisitHRule(hn *ast.HRuleNode) {
	tv.b.WriteString("(HR)")
	tv.visitAttributes(hn.Attrs)
}

var mapNestedListCode = map[ast.NestedListCode]string{
	ast.NestedListOrdered:   "(OL",
	ast.NestedListUnordered: "(UL",
	ast.NestedListQuote:     "(QL",
}

func (tv *TestVisitor) VisitNestedList(ln *ast.NestedListNode) {
	tv.b.WriteString(mapNestedListCode[ln.Code])
	for _, item := range ln.Items {
		tv.b.WriteString(" {")
		tv.visitItemSlice(item)
		tv.b.WriteByte('}')
	}
	tv.b.WriteByte(')')
}
func (tv *TestVisitor) VisitDescriptionList(dn *ast.DescriptionListNode) {
	tv.b.WriteString("(DL")
	for _, def := range dn.Descriptions {
		tv.b.WriteString(" (DT")
		tv.visitInlineSlice(def.Term)
		tv.b.WriteByte(')')
		for _, b := range def.Descriptions {
			tv.b.WriteString(" (DD ")
			tv.visitDescriptionSlice(b)
			tv.b.WriteByte(')')
		}
	}
	tv.b.WriteByte(')')
}

var alignString = map[ast.Alignment]string{
	ast.AlignDefault: "",
	ast.AlignLeft:    "l",
	ast.AlignCenter:  "c",
	ast.AlignRight:   "r",
}

// VisitTable emits a HTML table.
func (tv *TestVisitor) VisitTable(tn *ast.TableNode) {
	tv.b.WriteString("(TAB")
	if len(tn.Header) > 0 {
		tv.b.WriteString(" (TR")
		for _, cell := range tn.Header {
			tv.b.WriteString(" (TH")
			tv.b.WriteString(alignString[cell.Align])
			tv.visitInlineSlice(cell.Inlines)
			tv.b.WriteString(")")
		}
		tv.b.WriteString(")")
	}
	if len(tn.Rows) > 0 {
		tv.b.WriteString(" ")
		for _, row := range tn.Rows {
			tv.b.WriteString("(TR")
			for i, cell := range row {
				if i == 0 {
					tv.b.WriteString(" ")
				}
				tv.b.WriteString("(TD")
				tv.b.WriteString(alignString[cell.Align])
				tv.visitInlineSlice(cell.Inlines)
				tv.b.WriteString(")")
			}
			tv.b.WriteString(")")
		}
	}
	tv.b.WriteString(")")
}

func (tv *TestVisitor) VisitBLOB(bn *ast.BLOBNode) {
	tv.b.WriteString("(BLOB ")
	tv.b.WriteString(bn.Syntax)
	tv.b.WriteString(")")
}

func (tv *TestVisitor) VisitText(tn *ast.TextNode) {
	tv.b.WriteString(tn.Text)
}
func (tv *TestVisitor) VisitTag(tn *ast.TagNode) {
	tv.b.WriteByte('#')
	tv.b.WriteString(tn.Tag)
	tv.b.WriteByte('#')
}
func (tv *TestVisitor) VisitSpace(sn *ast.SpaceNode) {
	if len(sn.Lexeme) == 1 {
		tv.b.WriteString("SP")
	} else {
		fmt.Fprintf(&tv.b, "SP%d", len(sn.Lexeme))
	}
}
func (tv *TestVisitor) VisitBreak(bn *ast.BreakNode) {
	if bn.Hard {
		tv.b.WriteString("HB")
	} else {
		tv.b.WriteString("SB")
	}
}
func (tv *TestVisitor) VisitLink(tn *ast.LinkNode) {
	fmt.Fprintf(&tv.b, "(LINK %s", tn.Ref)
	tv.visitInlineSlice(tn.Inlines)
	tv.b.WriteByte(')')
	tv.visitAttributes(tn.Attrs)
}
func (tv *TestVisitor) VisitImage(in *ast.ImageNode) {
	fmt.Fprintf(&tv.b, "(IMAGE %s", in.Ref)
	tv.visitInlineSlice(in.Inlines)
	tv.b.WriteByte(')')
	tv.visitAttributes(in.Attrs)
}
func (tv *TestVisitor) VisitCite(cn *ast.CiteNode) {
	fmt.Fprintf(&tv.b, "(CITE %s", cn.Key)
	tv.visitInlineSlice(cn.Inlines)
	tv.b.WriteByte(')')
	tv.visitAttributes(cn.Attrs)
}
func (tv *TestVisitor) VisitFootnote(fn *ast.FootnoteNode) {
	tv.b.WriteString("(FN")
	tv.visitInlineSlice(fn.Inlines)
	tv.b.WriteByte(')')
	tv.visitAttributes(fn.Attrs)
}
func (tv *TestVisitor) VisitMark(mn *ast.MarkNode) {
	tv.b.WriteString("(MARK")
	if len(mn.Text) > 0 {
		tv.b.WriteByte(' ')
		tv.b.WriteString(mn.Text)
	}
	tv.b.WriteByte(')')
}

var mapCode = map[ast.FormatCode]rune{
	ast.FormatItalic:    '/',
	ast.FormatBold:      '*',
	ast.FormatUnder:     '_',
	ast.FormatStrike:    '~',
	ast.FormatMonospace: '\'',
	ast.FormatSuper:     '^',
	ast.FormatSub:       ',',
	ast.FormatQuote:     '"',
	ast.FormatQuotation: '<',
	ast.FormatSmall:     ';',
	ast.FormatSpan:      ':',
}

func (tv *TestVisitor) VisitFormat(fn *ast.FormatNode) {
	fmt.Fprintf(&tv.b, "{%c", mapCode[fn.Code])
	tv.visitInlineSlice(fn.Inlines)
	tv.b.WriteByte('}')
	tv.visitAttributes(fn.Attrs)
}

var mapLiteralCode = map[ast.LiteralCode]rune{
	ast.LiteralProg:    '`',
	ast.LiteralKeyb:    '+',
	ast.LiteralOutput:  '=',
	ast.LiteralComment: '%',
}

func (tv *TestVisitor) VisitLiteral(ln *ast.LiteralNode) {
	code, ok := mapLiteralCode[ln.Code]
	if !ok {
		panic(fmt.Sprintf("No element for code %v", ln.Code))
	}
	tv.b.WriteByte('{')
	tv.b.WriteRune(code)
	if len(ln.Text) > 0 {
		tv.b.WriteByte(' ')
		tv.b.WriteString(ln.Text)
	}
	tv.b.WriteByte('}')
	tv.visitAttributes(ln.Attrs)
}
func (tv *TestVisitor) visitBlockSlice(bns ast.BlockSlice) {
	for _, bn := range bns {
		bn.Accept(tv)
	}
}
func (tv *TestVisitor) visitItemSlice(ins ast.ItemSlice) {
	for _, in := range ins {
		in.Accept(tv)
	}
}
func (tv *TestVisitor) visitDescriptionSlice(dns ast.DescriptionSlice) {
	for _, dn := range dns {
		dn.Accept(tv)
	}
}
func (tv *TestVisitor) visitInlineSlice(ins ast.InlineSlice) {
	for _, in := range ins {
		tv.b.WriteByte(' ')
		in.Accept(tv)
	}
}
func (tv *TestVisitor) visitAttributes(a *ast.Attributes) {
	if a == nil || len(a.Attrs) == 0 {
		return
	}
	tv.b.WriteString("[ATTR")

	keys := make([]string, 0, len(a.Attrs))
	for k := range a.Attrs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		tv.b.WriteByte(' ')
		tv.b.WriteString(k)
		v := a.Attrs[k]
		if len(v) > 0 {
			tv.b.WriteByte('=')
			if strings.IndexRune(v, ' ') >= 0 {
				tv.b.WriteByte('"')
				tv.b.WriteString(v)
				tv.b.WriteByte('"')
			} else {
				tv.b.WriteString(v)
			}
		}
	}

	tv.b.WriteByte(']')
}
