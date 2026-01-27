// © Broadcom. All Rights Reserved.
// The term "Broadcom" refers to Broadcom Inc. and/or its subsidiaries.
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"testing"

	"bytes"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
)

func consumeString(s string) (out chan byte) {
	out = make(chan byte)
	go func() {
		for _, ch := range s {
			out <- byte(ch)
		}
		close(out)
	}()
	return
}

func collectIntoStringList(in chan string) (result []string) {
	for item := range in {
		result = append(result, item)
	}
	return
}

func uncommentFromString(s string) string {
	inCh := consumeString(s)
	out := uncomment(inCh)

	result := ""
	for item := range out {
		result += string(item)
	}
	return result
}

func TestParserUncomment(t *testing.T) {
	var result string

	test0 := "this is a straight-up line"
	result0 := test0

	result = uncommentFromString(test0)
	if result != result0 {
		t.Errorf("Expected %#v, received %#v", result0, result)
	}

	test1 := "this is a straight-up line with a newline\n"
	result1 := test1

	result = uncommentFromString(test1)
	if result != result1 {
		t.Errorf("Expected %#v, received %#v", result1, result)
	}

	test2 := "this line has a comment # at its end"
	result2 := "this line has a comment "

	result = uncommentFromString(test2)
	if result != result2 {
		t.Errorf("Expected %#v, received %#v", result2, result)
	}

	test3 := "# this whole line is commented"
	result3 := ""

	result = uncommentFromString(test3)
	if result != result3 {
		t.Errorf("Expected %#v, received %#v", result3, result)
	}

	test4 := "this\nhas\nmultiple\nlines"
	result4 := test4

	result = uncommentFromString(test4)
	if result != result4 {
		t.Errorf("Expected %#v, received %#v", result4, result)
	}

	test5 := "this only has\n# one line"
	result5 := "this only has\n"
	result = uncommentFromString(test5)
	if result != result5 {
		t.Errorf("Expected %#v, received %#v", result5, result)
	}

	test6 := "this is\npartially # commented"
	result6 := "this is\npartially "

	result = uncommentFromString(test6)
	if result != result6 {
		t.Errorf("Expected %#v, received %#v", result6, result)
	}

	test7 := "this # has\nmultiple # lines\ncommented # out"
	result7 := "this \nmultiple \ncommented "

	result = uncommentFromString(test7)
	if result != result7 {
		t.Errorf("Expected %#v, received %#v", result7, result)
	}
}

func tokenizeDhcpConfigFromString(s string) []string {
	inCh := consumeString(s)
	out := tokenizeDhcpConfig(inCh)

	result := make([]string, 0)
	for {
		item, ok := <-out
		if !ok {
			break
		}
		result = append(result, item)
	}
	return result
}

func TestParserTokenizeDhcp(t *testing.T) {

	test1 := `
subnet 127.0.0.0 netmask 255.255.255.252 {
    item 1234 5678;
	tabbed item-1 1234;
    quoted item-2 "hola mundo.";
}
`
	expected := []string{
		"subnet", "127.0.0.0", "netmask", "255.255.255.252", "{",
		"item", "1234", "5678", ";",
		"tabbed", "item-1", "1234", ";",
		"quoted", "item-2", "\"hola mundo.\"", ";",
		"}",
	}
	result := tokenizeDhcpConfigFromString(test1)

	t.Logf("testing for: %v", expected)
	t.Logf("checking out: %v", result)
	if len(result) != len(expected) {
		t.Fatalf("length of token lists do not match (%d != %d)", len(result), len(expected))
	}

	for index := range expected {
		if expected[index] != result[index] {
			t.Errorf("unexpected token at index %d: %v != %v", index, expected[index], result[index])
		}
	}
}

func consumeTokens(tokes []string) chan string {
	out := make(chan string)
	go func() {
		for _, item := range tokes {
			out <- item
		}
		out <- ";"
		close(out)
	}()
	return out
}

func TestParserDhcpParameters(t *testing.T) {
	var ch chan string

	test1 := []string{"option", "whee", "whooo"}
	ch = consumeTokens(test1)

	result := parseTokenParameter(ch)
	if result.name != "option" {
		t.Errorf("expected name %s, got %s", test1[0], result.name)
	}
	if len(result.operand) == 2 {
		if result.operand[0] != "whee" {
			t.Errorf("expected operand[%d] as %s, got %s", 0, "whee", result.operand[0])
		}
		if result.operand[1] != "whooo" {
			t.Errorf("expected operand[%d] as %s, got %s", 0, "whooo", result.operand[1])
		}
	} else {
		t.Errorf("expected %d operands, got %d", 2, len(result.operand))
	}

	test2 := []string{"whaaa", "whoaaa", ";", "wooops"}
	ch = consumeTokens(test2)

	result = parseTokenParameter(ch)
	if result.name != "whaaa" {
		t.Errorf("expected name %s, got %s", test2[0], result.name)
	}
	if len(result.operand) == 1 {
		if result.operand[0] != "whoaaa" {
			t.Errorf("expected operand[%d] as %s, got %s", 0, "whoaaa", result.operand[0])
		}
	} else {
		t.Errorf("expected %d operands, got %d", 1, len(result.operand))
	}

	test3 := []string{"optionz", "only", "{", "culled"}
	ch = consumeTokens(test3)

	result = parseTokenParameter(ch)
	if result.name != "optionz" {
		t.Errorf("expected name %s, got %s", test3[0], result.name)
	}
	if len(result.operand) == 1 {
		if result.operand[0] != "only" {
			t.Errorf("expected operand[%d] as %s, got %s", 0, "only", result.operand[0])
		}
	} else {
		t.Errorf("expected %d operands, got %d", 1, len(result.operand))
	}
}

func consumeDhcpConfig(items []string) (tkGroup, error) {
	out := make(chan string)
	tch := consumeTokens(items)

	go func() {
		for item := range tch {
			out <- item
		}
		close(out)
	}()

	return parseDhcpConfig(out)
}

func compareSlice(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestParserDhcpConfigParse(t *testing.T) {
	test1 := []string{
		"allow", "unused-option", ";",
		"lease-option", "1234", ";",
		"more", "options", "hi", ";",
	}
	result1, err := consumeDhcpConfig(test1)
	if err != nil {
		t.Fatalf("%s", err)
	}
	if len(result1.params) != 4 {
		t.Fatalf("expected %d params, got %d", 3, len(result1.params))
	}
	if result1.params[0].name != "allow" {
		t.Errorf("expected %s, got %s", "allow", result1.params[0].name)
	}
	if !compareSlice(result1.params[0].operand, []string{"unused-option"}) {
		t.Errorf("unexpected options parsed: %v", result1.params[0].operand)
	}
	if result1.params[1].name != "lease-option" {
		t.Errorf("expected %s, got %s", "lease-option", result1.params[1].name)
	}
	if !compareSlice(result1.params[1].operand, []string{"1234"}) {
		t.Errorf("unexpected options parsed: %v", result1.params[1].operand)
	}
	if result1.params[2].name != "more" {
		t.Errorf("expected %s, got %s", "lease-option", result1.params[2].name)
	}
	if !compareSlice(result1.params[2].operand, []string{"options", "hi"}) {
		t.Errorf("unexpected options parsed: %v", result1.params[2].operand)
	}

	test2 := []string{
		"first-option", ";",
		"child", "group", "{",
		"blah", ";",
		"meh", ";",
		"}",
		"hidden", "option", "57005", ";",
		"host", "device", "two", "{",
		"skipped", "option", ";",
		"more", "skipped", "options", ";",
		"}",
		"last", "option", "but", "unterminated",
	}
	result2, err := consumeDhcpConfig(test2)
	if err != nil {
		t.Fatalf("%s", err)
	}
	if len(result2.groups) != 2 {
		t.Fatalf("expected %d groups, got %d", 2, len(result2.groups))
	}

	if len(result2.params) != 3 {
		t.Errorf("expected %d options, got %d", 3, len(result2.params))
	}

	group0 := result2.groups[0]
	if group0.id.name != "child" {
		t.Errorf("expected group %s, got %s", "child", group0.id.name)
	}
	if len(group0.id.operand) != 1 {
		t.Errorf("expected group operand %d, got %d", 1, len(group0.params))
	}
	if len(group0.params) != 2 {
		t.Errorf("expected group params %d, got %d", 2, len(group0.params))
	}

	group1 := result2.groups[1]
	if group1.id.name != "host" {
		t.Errorf("expected group %s, got %s", "host", group1.id.name)
	}
	if len(group1.id.operand) != 2 {
		t.Errorf("expected group operand %d, got %d", 2, len(group1.params))
	}
	if len(group1.params) != 2 {
		t.Errorf("expected group params %d, got %d", 2, len(group1.params))
	}
}

func TestParserReadDhcpConfig(t *testing.T) {
	expected := []string{
		`{global}
grants : map[unknown-clients:0]
parameters : map[default-lease-time:1800 max-lease-time:7200]
`,

		`{subnet4 172.33.33.0/24},{global}
address : range4:172.33.33.128-172.33.33.254
options : map[broadcast-address:172.33.33.255 routers:172.33.33.2]
grants : map[unknown-clients:0]
parameters : map[default-lease-time:2400 max-lease-time:9600]
`,

		`{host name:vmnet8},{global}
address : hardware-address:ethernet[00:50:56:c0:00:08],fixed-address4:172.33.33.1
options : map[domain-name:"packer.test"]
grants : map[unknown-clients:0]
parameters : map[default-lease-time:1800 max-lease-time:7200]
`,
	}

	f, err := os.Open(filepath.Join("testdata", "dhcpd-example.conf"))
	if err != nil {
		t.Fatalf("Failed to open dhcpd.conf sample: %s", err)
	}
	defer f.Close()

	config, err := ReadDhcpConfiguration(f)
	if err != nil {
		t.Fatalf("Failed to read dhcpd.conf sample: %s", err)
	}

	if len(config) != 3 {
		t.Fatalf("expected %d entries, got %d", 3, len(config))
	}

	for index, item := range config {
		if item.repr() != expected[index] {
			t.Errorf("Parsing of declaration %d did not match what was expected", index)
			t.Logf("Result from parsing:\n%s", item.repr())
			t.Logf("Expected to parse:\n%s", expected[index])
		}
	}
}

func TestParserTokenizeNetworkMap(t *testing.T) {

	test1 := "group.attribute = \"string\""
	expected := []string{
		"group.attribute", "=", "\"string\"",
	}
	result := tokenizeDhcpConfigFromString(test1)
	if len(result) != len(expected) {
		t.Fatalf("length of token lists do not match (%d != %d)", len(result), len(expected))
	}

	for index := range expected {
		if expected[index] != result[index] {
			t.Errorf("unexpected token at index %d: %v != %v", index, expected[index], result[index])
		}
	}

	test2 := "attribute == \""
	expected = []string{
		"attribute", "==", "\"",
	}
	result = tokenizeDhcpConfigFromString(test2)
	if len(result) != len(expected) {
		t.Fatalf("length of token lists do not match (%d != %d)", len(result), len(expected))
	}

	test3 := "attribute ....... ======\nnew lines should make no difference"
	expected = []string{
		"attribute", ".......", "======", "new", "lines", "should", "make", "no", "difference",
	}
	result = tokenizeDhcpConfigFromString(test3)
	if len(result) != len(expected) {
		t.Fatalf("length of token lists do not match (%d != %d)", len(result), len(expected))
	}

	test4 := "\t\t\t\t    thishadwhitespacebeforebeingparsed\t \t \t \t\n\n"
	expected = []string{
		"thishadwhitespacebeforebeingparsed",
	}
	result = tokenizeDhcpConfigFromString(test4)
	if len(result) != len(expected) {
		t.Fatalf("length of token lists do not match (%d != %d)", len(result), len(expected))
	}
}

func TestParserReadNetworkMap(t *testing.T) {
	f, err := os.Open(filepath.Join("testdata", "netmap-example.conf"))
	if err != nil {
		t.Fatalf("Unable to open netmap.conf sample: %s", err)
	}
	defer f.Close()

	netmap, err := ReadNetworkMap(f)
	if err != nil {
		t.Fatalf("Unable to read netmap.conf sample: %s", err)
	}

	expectedKeys := []string{"device", "name"}
	for _, item := range netmap {
		for _, name := range expectedKeys {
			_, ok := item[name]
			if !ok {
				t.Errorf("unable to find expected key %v in map: %v", name, item)
			}
		}
	}

	expectedVmnet0 := [][]string{
		{"device", "vmnet0"},
		{"name", "Bridged"},
	}
	for _, item := range netmap {
		if item["device"] != "vmnet0" {
			continue
		}
		for _, expectPair := range expectedVmnet0 {
			name := expectPair[0]
			value := expectPair[1]
			if item[name] != value {
				t.Errorf("expected value %v for attribute %v, got %v", value, name, item[name])
			}
		}
	}

	expectedVmnet1 := [][]string{
		{"device", "vmnet1"},
		{"name", "HostOnly"},
	}
	for _, item := range netmap {
		if item["device"] != "vmnet1" {
			continue
		}
		for _, expectPair := range expectedVmnet1 {
			name := expectPair[0]
			value := expectPair[1]
			if item[name] != value {
				t.Errorf("expected value %v for attribute %v, got %v", value, name, item[name])
			}
		}
	}

	expectedVmnet8 := [][]string{
		{"device", "vmnet8"},
		{"name", "NAT"},
	}
	for _, item := range netmap {
		if item["device"] != "vmnet8" {
			continue
		}
		for _, expectPair := range expectedVmnet8 {
			name := expectPair[0]
			value := expectPair[1]
			if item[name] != value {
				t.Errorf("expected value %v for attribute %v, got %v", value, name, item[name])
			}
		}
	}
}

func collectIntoString(in chan byte) string {
	result := ""
	for item := range in {
		result += string(item)
	}
	return result
}

func TestParserConsumeUntilSentinel(t *testing.T) {

	test1 := "consume until a semicolon; yeh?"
	expected1 := "consume until a semicolon"

	ch := consumeString(test1)
	resultch, _ := consumeUntilSentinel(';', ch)
	result := string(resultch)
	if expected1 != result {
		t.Errorf("expected %#v, got %#v", expected1, result)
	}

	test2 := "; this is only a semi"
	expected2 := ""

	ch = consumeString(test2)
	resultch, _ = consumeUntilSentinel(';', ch)
	result = string(resultch)
	if expected2 != result {
		t.Errorf("expected %#v, got %#v", expected2, result)
	}
}

func TestParserFilterCharacters(t *testing.T) {

	test1 := []string{" ", "ignore all spaces"}
	expected1 := "ignoreallspaces"

	ch := consumeString(test1[1])
	outch := filterOutCharacters(bytes.NewBufferString(test1[0]).Bytes(), ch)
	result := collectIntoString(outch)
	if result != expected1 {
		t.Errorf("expected %#v, got %#v", expected1, result)
	}

	test2 := []string{"\n\v\t\r ", "ignore\nall\rwhite\v\v space                "}
	expected2 := "ignoreallwhitespace"

	ch = consumeString(test2[1])
	outch = filterOutCharacters(bytes.NewBufferString(test2[0]).Bytes(), ch)
	result = collectIntoString(outch)
	if result != expected2 {
		t.Errorf("expected %#v, got %#v", expected2, result)
	}
}

func TestParserConsumeOpenClosePair(t *testing.T) {
	test1 := "(everything)"
	expected1 := []string{"", test1}

	testch := consumeString(test1)
	prefix, ch := consumeOpenClosePair('(', ')', testch)
	if string(prefix) != expected1[0] {
		t.Errorf("expected prefix %#v, got %#v", expected1[0], prefix)
	}
	result := collectIntoString(ch)
	if result != expected1[1] {
		t.Errorf("expected %#v, got %#v", expected1[1], test1)
	}

	test2 := "prefixed (everything)"
	expected2 := []string{"prefixed ", "(everything)"}

	testch = consumeString(test2)
	prefix, ch = consumeOpenClosePair('(', ')', testch)
	if string(prefix) != expected2[0] {
		t.Errorf("expected prefix %#v, got %#v", expected2[0], prefix)
	}
	result = collectIntoString(ch)
	if result != expected2[1] {
		t.Errorf("expected %#v, got %#v", expected2[1], test2)
	}

	test3 := "this(is()suffixed"
	expected3 := []string{"this", "(is()"}

	testch = consumeString(test3)
	prefix, ch = consumeOpenClosePair('(', ')', testch)
	if string(prefix) != expected3[0] {
		t.Errorf("expected prefix %#v, got %#v", expected3[0], prefix)
	}
	result = collectIntoString(ch)
	if result != expected3[1] {
		t.Errorf("expected %#v, got %#v", expected3[1], test2)
	}
}

func TestParserCombinators(t *testing.T) {

	test1 := "across # ignore\nmultiple lines;"
	expected1 := "across multiple lines"

	ch := consumeString(test1)
	inch := uncomment(ch)
	whch := filterOutCharacters([]byte{'\n'}, inch)
	resultch, _ := consumeUntilSentinel(';', whch)
	result := string(resultch)
	if expected1 != result {
		t.Errorf("expected %#v, got %#v", expected1, result)
	}

	test2 := "lease blah {\n    blah\r\n# skipping this line\nblahblah  # ignore semicolon;\n last item;\n\n };;;;;;"
	expected2 := []string{"lease blah ", "{    blahblahblah   last item; }"}

	ch = consumeString(test2)
	inch = uncomment(ch)
	whch = filterOutCharacters([]byte{'\n', '\v', '\r'}, inch)
	prefix, pairch := consumeOpenClosePair('{', '}', whch)

	result = collectIntoString(pairch)
	if string(prefix) != expected2[0] {
		t.Errorf("expected prefix %#v, got %#v", expected2[0], prefix)
	}
	if result != expected2[1] {
		t.Errorf("expected %#v, got %#v", expected2[1], result)
	}

	test3 := "lease blah { # comment\n item 1;\n item 2;\n } not imortant"
	expected3Prefix := "lease blah "
	expected3 := []string{"{  item 1", " item 2", " }"}

	sch := consumeString(test3)
	inch = uncomment(sch)
	wch := filterOutCharacters([]byte{'\n', '\v', '\r'}, inch)
	lease, itemch := consumeOpenClosePair('{', '}', wch)
	if string(lease) != expected3Prefix {
		t.Errorf("expected %#v, got %#v", expected3Prefix, string(lease))
	}

	var result3 []string
	for reading := true; reading; {
		item, ok := consumeUntilSentinel(';', itemch)
		result3 = append(result3, string(item))
		if !ok {
			reading = false
		}
	}

	for index := range expected3 {
		if expected3[index] != result3[index] {
			t.Errorf("expected index %d as %#v, got %#v", index, expected3[index], result3[index])
		}
	}
}

func TestParserDhcpdLeaseBytesDecoder(t *testing.T) {
	test1 := "00:0d:0e:0a:0d:00"
	expected1 := []byte{0, 13, 14, 10, 13, 0}

	result, err := decodeDhcpdLeaseBytes(test1)
	if err != nil {
		t.Errorf("unable to decode address: %s", err)
	}
	if !bytes.Equal(result, expected1) {
		t.Errorf("expected %v, got %v", expected1, result)
	}

	test2 := "11"
	expected2 := []byte{17}

	result, err = decodeDhcpdLeaseBytes(test2)
	if err != nil {
		t.Errorf("unable to decode address: %s", err)
	}
	if !bytes.Equal(result, expected2) {
		t.Errorf("expected %v, got %v", expected2, result)
	}

	failtest1 := ""
	_, err = decodeDhcpdLeaseBytes(failtest1)
	if err == nil {
		t.Errorf("expected decoding error: %s", err)
	}

	failtest2 := "000000"
	_, err = decodeDhcpdLeaseBytes(failtest2)
	if err == nil {
		t.Errorf("expected decoding error: %s", err)
	}

	failtest3 := "000:00"
	_, err = decodeDhcpdLeaseBytes(failtest3)
	if err == nil {
		t.Errorf("expected decoding error: %s", err)
	}

	failtest4 := "00:00:"
	_, err = decodeDhcpdLeaseBytes(failtest4)
	if err == nil {
		t.Errorf("expected decoding error: %s", err)
	}
}

func consumeLeaseString(s string) chan byte {
	sch := consumeString(s)
	uncommentedch := uncomment(sch)
	return filterOutCharacters([]byte{'\n', '\r', '\v'}, uncommentedch)
}

func TestParserReadDhcpdLeaseEntry(t *testing.T) {
	test1 := "lease 127.0.0.1 {\nhardware ethernet 00:11:22:33  ;\nuid 00:11  ;\n }"
	expected1 := map[string]string{
		"address": "127.0.0.1",
		"ether":   "00112233",
		"uid":     "0011",
	}

	result, err := readDhcpdLeaseEntry(consumeLeaseString(test1))
	if err != nil {
		t.Errorf("error parsing entry: %s", err)
	}
	if result.address != expected1["address"] {
		t.Errorf("expected address %v, got %v", expected1["address"], result.address)
	}
	if hex.EncodeToString(result.ether) != expected1["ether"] {
		t.Errorf("expected ether %v, got %v", expected1["ether"], hex.EncodeToString(result.ether))
	}
	if hex.EncodeToString(result.uid) != expected1["uid"] {
		t.Errorf("expected uid %v, got %v", expected1["uid"], hex.EncodeToString(result.uid))
	}

	test2 := "  \n\t lease 192.168.21.254{ hardware\n   ethernet 44:55:66:77:88:99;uid 00:1\n1:22:3\r3:44;\n starts 57005 2006/01/02 15:04:05;ends 57005 2006/01/03 15:04:05;\tunknown item1; unknown item2;  }     "
	expected2 := map[string]string{
		"address": "192.168.21.254",
		"ether":   "445566778899",
		"uid":     "0011223344",
		"starts":  "2006-01-02 15:04:05 +0000 UTC",
		"ends":    "2006-01-03 15:04:05 +0000 UTC",
	}
	result, err = readDhcpdLeaseEntry(consumeLeaseString(test2))
	if err != nil {
		t.Errorf("error parsing entry: %s", err)
	}
	if result.address != expected2["address"] {
		t.Errorf("expected address %v, got %v", expected2["address"], result.address)
	}
	if hex.EncodeToString(result.ether) != expected2["ether"] {
		t.Errorf("expected ether %v, got %v", expected2["ether"], hex.EncodeToString(result.ether))
	}
	if hex.EncodeToString(result.uid) != expected2["uid"] {
		t.Errorf("expected uid %v, got %v", expected2["uid"], hex.EncodeToString(result.uid))
	}
	if result.starts.String() != expected2["starts"] {
		t.Errorf("expected starts %v, got %v", expected2["starts"], result.starts)
	}
	if result.ends.String() != expected2["ends"] {
		t.Errorf("expected ends %v, got %v", expected2["ends"], result.ends)
	}
	if result.startsWeekday != 57005 {
		t.Errorf("expected starts weekday %v, got %v", 57005, result.startsWeekday)
	}
	if result.endsWeekday != 57005 {
		t.Errorf("expected ends weekday %v, got %v", 57005, result.endsWeekday)
	}
}

func TestParserReadDhcpdLeases(t *testing.T) {
	f, err := os.Open(filepath.Join("testdata", "dhcpd-example.leases"))
	if err != nil {
		t.Fatalf("Unable to open dhcpd.leases sample: %s", err)
	}
	defer f.Close()

	results, err := ReadDhcpdLeaseEntries(f)
	if err != nil {
		t.Fatalf("Error reading lease: %s", err)
	}

	// some simple utilities
	filterAddress := func(address string, items []dhcpLeaseEntry) (result []dhcpLeaseEntry) {
		for _, item := range items {
			if item.address == address {
				result = append(result, item)
			}
		}
		return
	}

	findUid := func(uid string, items []dhcpLeaseEntry) *dhcpLeaseEntry {
		for _, item := range items {
			if uid == hex.EncodeToString(item.uid) {
				return &item
			}
		}
		return nil
	}

	findEther := func(ether string, items []dhcpLeaseEntry) *dhcpLeaseEntry {
		for _, item := range items {
			if ether == hex.EncodeToString(item.ether) {
				return &item
			}
		}
		return nil
	}

	// actual unit tests
	test1 := map[string]string{
		"address": "127.0.0.19",
		"uid":     "010dead099aabb",
		"ether":   "0dead099aabb",
	}
	test1Findings := filterAddress(test1["address"], results)
	if len(test1Findings) != 2 {
		t.Errorf("expected %d matching entries, got %d", 2, len(test1Findings))
	} else {
		res := findEther(test1["ether"], test1Findings)
		if res == nil {
			t.Errorf("unable to find item with ether %v", test1["ether"])
		} else if hex.EncodeToString(res.uid) != test1["uid"] {
			t.Errorf("expected uid %s, got %s", test1["uid"], hex.EncodeToString(res.uid))
		}
	}

	test2 := map[string]string{
		"address": "127.0.0.19",
		"uid":     "010dead0667788",
		"ether":   "0dead0667788",
	}
	test2Findings := filterAddress(test2["address"], results)
	if len(test2Findings) != 2 {
		t.Errorf("expected %d matching entries, got %d", 2, len(test2Findings))
	} else {
		res := findEther(test2["ether"], test2Findings)
		if res == nil {
			t.Errorf("unable to find item with ether %v", test2["ether"])
		} else if hex.EncodeToString(res.uid) != test2["uid"] {
			t.Errorf("expected uid %s, got %s", test2["uid"], hex.EncodeToString(res.uid))
		}
	}

	test3 := map[string]string{
		"address": "127.0.0.17",
		"uid":     "010dead0334455",
		"ether":   "0dead0667788",
	}
	test3Findings := filterAddress(test3["address"], results)
	if len(test3Findings) != 2 {
		t.Errorf("expected %d matching entries, got %d", 2, len(test3Findings))
	} else {
		res := findUid(test3["uid"], test3Findings)
		if res == nil {
			t.Errorf("unable to find item with uid %v", test3["uid"])
		} else if hex.EncodeToString(res.ether) != test3["ether"] {
			t.Errorf("expected ethernet hardware %s, got %s", test3["ether"], hex.EncodeToString(res.ether))
		}
	}

	test4 := map[string]string{
		"address": "127.0.0.17",
		"uid":     "010dead0001122",
		"ether":   "0dead0667788",
	}
	test4Findings := filterAddress(test4["address"], results)
	if len(test4Findings) != 2 {
		t.Errorf("expected %d matching entries, got %d", 2, len(test4Findings))
	} else {
		res := findUid(test4["uid"], test4Findings)
		if res == nil {
			t.Errorf("unable to find item with uid %v", test4["uid"])
		} else if hex.EncodeToString(res.ether) != test4["ether"] {
			t.Errorf("expected ethernet hardware %s, got %s", test4["ether"], hex.EncodeToString(res.ether))
		}
	}
}

func consumeAppleLeaseString(s string) chan byte {
	sch := consumeString(s)
	uncommentedch := uncomment(sch)
	return filterOutCharacters([]byte{'\r', '\v'}, uncommentedch)
}

func TestParserReadAppleDhcpdLeaseEntry(t *testing.T) {
	test1 := `{
		ip_address=192.168.111.3
		hw_address=1,0:c:56:3c:e7:22
		identifier=1,0:c:56:3c:e7:22
		lease=0x5fd78ae2
		name=vagrant-2019
		fake=field
	}`
	expected1 := map[string]string{
		"ipAddress": "192.168.111.3",
		"hwAddress": "000c563ce722",
		"id":        "000c563ce722",
		"lease":     "0x5fd78ae2",
		"name":      "vagrant-2019",
	}
	expectedExtra1 := map[string]string{
		"fake": "field",
	}

	result, err := readAppleDhcpdLeaseEntry(consumeAppleLeaseString(test1))
	if err != nil {
		t.Errorf("error parsing entry: %s", err)
	}
	if result.ipAddress != expected1["ipAddress"] {
		t.Errorf("expected ipAddress %v, got %v", expected1["ipAddress"], result.ipAddress)
	}
	if hex.EncodeToString(result.hwAddress) != expected1["hwAddress"] {
		t.Errorf("expected hwAddress %v, got %v", expected1["hwAddress"], hex.EncodeToString(result.hwAddress))
	}
	if hex.EncodeToString(result.id) != expected1["id"] {
		t.Errorf("expected id %v, got %v", expected1["id"], hex.EncodeToString(result.id))
	}
	if result.lease != expected1["lease"] {
		t.Errorf("expected lease %v, got %v", expected1["lease"], result.lease)
	}
	if result.name != expected1["name"] {
		t.Errorf("expected name %v, got %v", expected1["name"], result.name)
	}
	if result.extra["fake"] != expectedExtra1["fake"] {
		t.Errorf("expected extra %v, got %v", expectedExtra1["fake"], result.extra["fake"])
	}

	test2 := `{
		ip_address=192.168.111.4
		hw_address=1,0:c:56:3c:e7:23
		identifier=1,0:c:56:3c:e7:23
	}`
	expected2 := map[string]string{
		"ipAddress": "192.168.111.4",
		"hwAddress": "000c563ce723",
		"id":        "000c563ce723",
	}

	result, err = readAppleDhcpdLeaseEntry(consumeAppleLeaseString(test2))
	if err != nil {
		t.Errorf("error parsing entry: %s", err)
	}
	if result.ipAddress != expected2["ipAddress"] {
		t.Errorf("expected ipAddress %v, got %v", expected2["ipAddress"], result.ipAddress)
	}
	if hex.EncodeToString(result.hwAddress) != expected2["hwAddress"] {
		t.Errorf("expected hwAddress %v, got %v", expected2["hwAddress"], hex.EncodeToString(result.hwAddress))
	}
	if hex.EncodeToString(result.id) != expected2["id"] {
		t.Errorf("expected id %v, got %v", expected2["id"], hex.EncodeToString(result.id))
	}
}

func TestParserReadAppleDhcpdLeases(t *testing.T) {
	f, err := os.Open(filepath.Join("testdata", "apple-dhcpd-example.leases"))
	if err != nil {
		t.Fatalf("Unable to open dhcpd.leases sample: %s", err)
	}
	defer f.Close()

	results, err := ReadAppleDhcpdLeaseEntries(f)
	if err != nil {
		t.Fatalf("Error reading lease: %s", err)
	}

	// some simple utilities
	filterIpAddress := func(ipAddress string, items []appleDhcpLeaseEntry) (result []appleDhcpLeaseEntry) {
		for _, item := range items {
			if item.ipAddress == ipAddress {
				result = append(result, item)
			}
		}
		return
	}

	findId := func(id string, items []appleDhcpLeaseEntry) *appleDhcpLeaseEntry {
		for _, item := range items {
			if id == hex.EncodeToString(item.id) {
				return &item
			}
		}
		return nil
	}

	findHardwareAddress := func(hwAddr string, items []appleDhcpLeaseEntry) *appleDhcpLeaseEntry {
		for _, item := range items {
			if hwAddr == hex.EncodeToString(item.hwAddress) {
				return &item
			}
		}
		return nil
	}

	// actual unit tests
	test1 := map[string]string{
		"ipAddress": "127.0.0.19",
		"id":        "0dead099aabb",
		"hwAddress": "0dead099aabb",
	}
	test1Findings := filterIpAddress(test1["ipAddress"], results)
	if len(test1Findings) != 2 {
		t.Errorf("expected %d matching entries, got %d", 2, len(test1Findings))
	} else {
		res := findHardwareAddress(test1["hwAddress"], test1Findings)
		if res == nil {
			t.Errorf("unable to find item with hwAddress %v", test1["hwAddress"])
		} else if hex.EncodeToString(res.id) != test1["id"] {
			t.Errorf("expected id %s, got %s", test1["id"], hex.EncodeToString(res.id))
		}
	}

	test2 := map[string]string{
		"ipAddress": "127.0.0.19",
		"id":        "0dead0667788",
		"hwAddress": "0dead0667788",
	}
	test2Findings := filterIpAddress(test2["ipAddress"], results)
	if len(test2Findings) != 2 {
		t.Errorf("expected %d matching entries, got %d", 2, len(test2Findings))
	} else {
		res := findHardwareAddress(test2["hwAddress"], test2Findings)
		if res == nil {
			t.Errorf("unable to find item with hwAddress %v", test2["hwAddress"])
		} else if hex.EncodeToString(res.id) != test2["id"] {
			t.Errorf("expected id %s, got %s", test2["id"], hex.EncodeToString(res.id))
		}
	}

	test3 := map[string]string{
		"ipAddress": "127.0.0.17",
		"id":        "0dead0334455",
		"hwAddress": "0dead0667788",
	}
	test3Findings := filterIpAddress(test3["ipAddress"], results)
	if len(test3Findings) != 2 {
		t.Errorf("expected %d matching entries, got %d", 2, len(test3Findings))
	} else {
		res := findId(test3["id"], test3Findings)
		if res == nil {
			t.Errorf("unable to find item with id %v", test3["id"])
		} else if hex.EncodeToString(res.hwAddress) != test3["hwAddress"] {
			t.Errorf("expected hardware address %s, got %s", test3["hwAddress"], hex.EncodeToString(res.hwAddress))
		}
	}

	test4 := map[string]string{
		"ipAddress": "127.0.0.17",
		"id":        "0dead0001122",
		"hwAddress": "0dead0667788",
	}
	test4Findings := filterIpAddress(test4["ipAddress"], results)
	if len(test4Findings) != 2 {
		t.Errorf("expected %d matching entries, got %d", 2, len(test4Findings))
	} else {
		res := findId(test4["id"], test4Findings)
		if res == nil {
			t.Errorf("unable to find item with id %v", test4["id"])
		} else if hex.EncodeToString(res.hwAddress) != test4["hwAddress"] {
			t.Errorf("expected hardware address %s, got %s", test4["hwAddress"], hex.EncodeToString(res.hwAddress))
		}
	}

	test5 := map[string]string{
		"ipAddress": "127.0.0.20",
		"id":        "0dead099aabc",
		"hwAddress": "0dead099aabc",
	}
	test5Findings := filterIpAddress(test5["ipAddress"], results)
	if len(test5Findings) != 1 {
		t.Errorf("expected %d matching entries, got %d", 1, len(test5Findings))
	} else {
		res := findId(test5["id"], test5Findings)
		if res == nil {
			t.Errorf("unable to find item with id %v", test5["id"])
		} else if hex.EncodeToString(res.hwAddress) != test5["hwAddress"] {
			t.Errorf("expected hardware address %s, got %s", test5["hwAddress"], hex.EncodeToString(res.hwAddress))
		}
	}
}

func TestParserTokenizeNetworkingConfig(t *testing.T) {
	tests := []string{
		"words       words       words",
		"newlines\n\n\n\n\n\n\n\nnewlines\r\r\r\r\r\r\r\rnewlines\n\n\n\n",
		"       newline-less",
	}
	expects := [][]string{
		{"words", "words", "words"},
		{"newlines", "\n", "newlines", "\n", "newlines", "\n"},
		{"newline-less"},
	}

	for testnum := 0; testnum < len(tests); testnum++ {
		inCh := consumeString(tests[testnum])
		outCh := tokenizeNetworkingConfig(inCh)
		result := collectIntoStringList(outCh)

		expected := expects[testnum]
		if len(result) != len(expected) {
			t.Errorf("test %d expected %d items, got %d instead", 1+testnum, len(expected), len(result))
			continue
		}

		ok := true
		for index := 0; index < len(expected); index++ {
			if result[index] != expected[index] {
				ok = false
			}
		}
		if !ok {
			t.Errorf("test %d expected %#v, got %#v", 1+testnum, expects[testnum], result)
		}
	}
}

func TestParserSplitNetworkingConfig(t *testing.T) {
	tests := []string{
		"this is a story\n\n\nabout some newlines",
		"\n\n\nthat can begin and end with newlines\n\n\n",
		"   in\n\n\nsome\ncases\nit\ncan\nend\nwith\nan\nempty\nstring\n\n\n\n",
		"\n\n\nand\nbegin\nwith\nan\nempty\nstring   ",
	}
	expects := [][]string{
		{"this is a story", "about some newlines"},
		{"that can begin and end with newlines"},
		{"in", "some", "cases", "it", "can", "end", "with", "an", "empty", "string"},
		{"and", "begin", "with", "an", "empty", "string"},
	}

	for testnum := 0; testnum < len(tests); testnum++ {
		inCh := consumeString(tests[testnum])
		stringCh := tokenizeNetworkingConfig(inCh)
		outCh := splitNetworkingConfig(stringCh)

		result := make([]string, 0)
		for item := range outCh {
			result = append(result, strings.Join(item, " "))
		}

		expected := expects[testnum]
		if len(result) != len(expected) {
			t.Errorf("test %d expected %d items, got %d instead", 1+testnum, len(expected), len(result))
			continue
		}

		ok := true
		for index := 0; index < len(expected); index++ {
			if result[index] != expected[index] {
				ok = false
			}
		}
		if !ok {
			t.Errorf("test %d expected %#v, got %#v", 1+testnum, expects[testnum], result)
		}
	}
}

func TestParserParseNetworkingConfigVersion(t *testing.T) {
	successTests := []string{"VERSION=4,2"}
	failureTests := []string{
		"VERSION=1=2",
		"VERSION=3,4,5",
		"VERSION=a,b",
	}

	for testnum := 0; testnum < len(successTests); testnum++ {
		test := []string{successTests[testnum]}
		if _, err := networkingReadVersion(test); err != nil {
			t.Errorf("success-test %d parsing failed: %v", 1+testnum, err)
		}
	}

	for testnum := 0; testnum < len(successTests); testnum++ {
		test := []string{failureTests[testnum]}
		if _, err := networkingReadVersion(test); err == nil {
			t.Errorf("failure-test %d should have failed", 1+testnum)
		}
	}
}

func TestParserParseNetworkingConfigEntries(t *testing.T) {
	tests := []string{
		"answer VNET_999_ANYTHING option",
		"remove_answer VNET_123_ALSOANYTHING",
		"add_nat_portfwd 24 udp 42 127.0.0.1 24",
		"remove_nat_portfwd 42 tcp 2502",
		"add_dhcp_mac_to_ip 57005 00:0d:0e:0a:0d:00 127.0.0.2",
		"remove_dhcp_mac_to_ip 57005 00:0d:0e:0a:0d:00",
		"add_bridge_mapping string 51",
		"remove_bridge_mapping string",
		"add_nat_prefix 57005 /24",
		"remove_nat_prefix 57005 /31",
	}

	for testnum := 0; testnum < len(tests); testnum++ {
		test := strings.Split(tests[testnum], " ")
		parser := NetworkingParserByCommand(test[0])
		if parser == nil {
			t.Errorf("test %d unable to parse command: %#v", 1+testnum, test)
			continue
		}
		operandParser := *parser

		_, err := operandParser(test[1:])
		if err != nil {
			t.Errorf("test %d unable to parse command parameters %#v: %v", 1+testnum, test, err)
		}
	}
}

func TestParserReadNetworingConfig(t *testing.T) {
	expectedAnswerVnet1 := map[string]string{
		"DHCP":             "yes",
		"DHCP_CFG_HASH":    "01F4CE0D79A1599698B6E5814CCB68058BB0ED5E",
		"HOSTONLY_NETMASK": "255.255.255.0",
		"HOSTONLY_SUBNET":  "192.168.70.0",
		"NAT":              "no",
		"VIRTUAL_ADAPTER":  "yes",
	}

	f, err := os.Open(filepath.Join("testdata", "networking-example"))
	if err != nil {
		t.Fatalf("Unable to open networking-example sample: %s", err)
	}
	defer f.Close()

	config, err := ReadNetworkingConfig(f)
	if err != nil {
		t.Fatalf("error parsing networking-example: %s", err)
	}

	if vnet, ok := config.answer[1]; ok {
		for answerKey := range expectedAnswerVnet1 {
			result, ok := vnet[answerKey]
			if !ok {
				t.Errorf("unable to find key %s in VNET_%d answer", answerKey, 1)
				continue
			}

			if result != expectedAnswerVnet1[answerKey] {
				t.Errorf("expected key %s for VNET_%d to be %v, got %v", answerKey, 1, expectedAnswerVnet1[answerKey], result)
			}
		}

	} else {
		t.Errorf("unable to find VNET_%d answer", 1)
	}

	expectedAnswerVnet8 := map[string]string{
		"DHCP":             "yes",
		"DHCP_CFG_HASH":    "C30F14F65A0FE4B5DCC6C67497D7A8A33E5E538C",
		"HOSTONLY_NETMASK": "255.255.255.0",
		"HOSTONLY_SUBNET":  "172.16.41.0",
		"NAT":              "yes",
		"VIRTUAL_ADAPTER":  "yes",
	}

	if vnet, ok := config.answer[8]; ok {
		for answerKey := range expectedAnswerVnet8 {
			result, ok := vnet[answerKey]
			if !ok {
				t.Errorf("unable to find key %s in VNET_%d answer", answerKey, 8)
				continue
			}

			if result != expectedAnswerVnet8[answerKey] {
				t.Errorf("expected key %s for VNET_%d to be %v, got %v", answerKey, 8, expectedAnswerVnet8[answerKey], result)
			}
		}

	} else {
		t.Errorf("unable to find VNET_%d answer", 8)
	}

	expectedNatPortFwd8 := map[string]string{
		"tcp/2200":  "172.16.41.129:3389",
		"tcp/2201":  "172.16.41.129:3389",
		"tcp/2222":  "172.16.41.129:22",
		"tcp/3389":  "172.16.41.131:3389",
		"tcp/55985": "172.16.41.129:5985",
		"tcp/55986": "172.16.41.129:5986",
	}

	if vnet, ok := config.natPortFwd[8-1]; ok {
		for natKey := range expectedNatPortFwd8 {
			result, ok := vnet[natKey]
			if !ok {
				t.Errorf("unable to find key %s in VNET_%d nat_portfwd", natKey, 8)
				continue
			}

			if result != expectedNatPortFwd8[natKey] {
				t.Errorf("expected key %s for VNET_%d to be %v, got %v", natKey, 8, expectedNatPortFwd8[natKey], result)
			}
		}
	} else {
		t.Errorf("unable to find VNET_%d answer", 8-1)
	}
}
