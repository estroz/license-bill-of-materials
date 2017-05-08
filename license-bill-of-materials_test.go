package main

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type testResult struct {
	Package  string
	Licenses []*testResultLicense
	Err      string
}

type testResultLicense struct {
	License string
	Score   int
	Extra   int
	Missing int
}

func listTestLicenses(pkgs []string) ([]testResult, error) {
	gopath, err := filepath.Abs("testdata")
	if err != nil {
		return nil, err
	}
	licenses, err := listLicenses(gopath, pkgs)
	if err != nil {
		return nil, err
	}
	res := []testResult{}
	for _, l := range licenses {
		r := testResult{
			Package: l.Package,
		}
		trls := []*testResultLicense{}
		for _, li := range l.LicenseInfos {
			trl := testResultLicense{}
			if li.Template != nil {
				trl.License = li.Template.Title
				trl.Score = int(100 * li.Score)
			}
			trl.Extra = len(li.ExtraWords)
			trl.Missing = len(li.MissingWords)
			trls = append(trls, &trl)
		}
		r.Licenses = trls
		if l.Err != "" {
			r.Err = "some error"
		}
		res = append(res, r)
	}
	return res, nil
}

func compareTestLicenses(pkgs []string, wanted []testResult) error {
	stringify := func(res []testResult) string {
		parts := []string{}
		for _, r := range res {
			s := fmt.Sprintf("%s:", r.Package)
			for i, trl := range r.Licenses {
				if i > 0 {
					s += ";"
				}
				s += fmt.Sprintf(" \"%s\" %d%%", trl.License, trl.Score)
				if r.Err != "" {
					s += " " + r.Err
				}
				if trl.Extra > 0 {
					s += fmt.Sprintf(" +%d", trl.Extra)
				}
				if trl.Missing > 0 {
					s += fmt.Sprintf(" -%d", trl.Missing)
				}
			}
			parts = append(parts, s)
		}
		return strings.Join(parts, "\n")
	}

	licenses, err := listTestLicenses(pkgs)
	if err != nil {
		return err
	}
	got := stringify(licenses)
	expected := stringify(wanted)
	if got != expected {
		return fmt.Errorf("licenses do not match:\n%s\n!=\n%s", got, expected)
	}
	return nil
}

func TestNoDependencies(t *testing.T) {
	err := compareTestLicenses([]string{"colors/red"}, []testResult{
		{Package: "colors/red", Licenses: []*testResultLicense{
			{License: "MIT License", Score: 98, Missing: 2},
		},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}

// Multiple licenses should be detected
func TestMultipleLicenses(t *testing.T) {
	err := compareTestLicenses([]string{"colors/blue"}, []testResult{
		{Package: "colors/blue", Licenses: []*testResultLicense{
			{License: "MIT License", Score: 98, Missing: 2},
			{License: "Apache License 2.0", Score: 100}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestNoLicense(t *testing.T) {
	err := compareTestLicenses([]string{"colors/green"}, []testResult{
		{Package: "colors/green", Licenses: []*testResultLicense{
			{License: "", Score: 0}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestMainWithDependencies(t *testing.T) {
	// It also tests license retrieval in parent directory.
	err := compareTestLicenses([]string{"colors/cmd/paint"}, []testResult{
		{Package: "colors/cmd/paint", Licenses: []*testResultLicense{
			{License: "Academic Free License v3.0", Score: 100}},
		},
		{Package: "colors/red", Licenses: []*testResultLicense{
			{License: "MIT License", Score: 98, Missing: 2}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestMainWithAliasedDependencies(t *testing.T) {
	err := compareTestLicenses([]string{"colors/cmd/mix"}, []testResult{
		{Package: "colors/cmd/mix", Licenses: []*testResultLicense{
			{License: "Academic Free License v3.0", Score: 100}},
		},
		{Package: "colors/red", Licenses: []*testResultLicense{
			{License: "MIT License", Score: 98, Missing: 2}},
		},
		{Package: "couleurs/red", Licenses: []*testResultLicense{
			{License: "GNU Lesser General Public License v2.1", Score: 100}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestMissingPackage(t *testing.T) {
	_, err := listTestLicenses([]string{"colors/missing"})
	if err == nil {
		t.Fatal("no error on missing package")
	}
	if _, ok := err.(*MissingError); !ok {
		t.Fatalf("MissingError expected")
	}
}

func TestMismatch(t *testing.T) {
	err := compareTestLicenses([]string{"colors/yellow"}, []testResult{
		{Package: "colors/yellow", Licenses: []*testResultLicense{
			{License: "Microsoft Reciprocal License", Score: 25, Extra: 106,
				Missing: 131}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestNoBuildableGoSourceFiles(t *testing.T) {
	_, err := listTestLicenses([]string{"colors/cmd"})
	if err == nil {
		t.Fatal("no error on missing package")
	}
	if _, ok := err.(*MissingError); !ok {
		t.Fatalf("MissingError expected")
	}
}

func TestBroken(t *testing.T) {
	err := compareTestLicenses([]string{"colors/broken"}, []testResult{
		{Package: "colors/broken", Licenses: []*testResultLicense{
			{License: "GNU General Public License v3.0", Score: 100}},
		},
		{Package: "colors/missing", Err: "some error", Licenses: []*testResultLicense{
			{License: "", Score: 0}},
		},
		{Package: "colors/red", Licenses: []*testResultLicense{
			{License: "MIT License", Score: 98, Missing: 2}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestBrokenDependency(t *testing.T) {

	err := compareTestLicenses([]string{"colors/purple"}, []testResult{
		{Package: "colors/broken", Licenses: []*testResultLicense{
			{License: "GNU General Public License v3.0", Score: 100}},
		},
		{Package: "colors/missing", Err: "some error", Licenses: []*testResultLicense{
			{License: "", Score: 0}},
		},
		{Package: "colors/purple", Licenses: []*testResultLicense{
			{License: "", Score: 0}},
		},
		{Package: "colors/red", Licenses: []*testResultLicense{
			{License: "MIT License", Score: 98, Missing: 2}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestPackageExpression(t *testing.T) {
	err := compareTestLicenses([]string{"colors/cmd/..."}, []testResult{
		{Package: "colors/cmd/mix", Licenses: []*testResultLicense{
			{License: "Academic Free License v3.0", Score: 100}},
		},
		{Package: "colors/cmd/paint", Licenses: []*testResultLicense{
			{License: "Academic Free License v3.0", Score: 100}},
		},
		{Package: "colors/red", Licenses: []*testResultLicense{
			{License: "MIT License", Score: 98, Missing: 2}},
		},
		{Package: "couleurs/red", Licenses: []*testResultLicense{
			{License: "GNU Lesser General Public License v2.1", Score: 100}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestCleanLicenseData(t *testing.T) {
	data := `The MIT License (MIT)

	Copyright (c) 2013 Ben Johnson

	Some other lines.
	And more.
	`
	cleaned := string(cleanLicenseData([]byte(data)))
	wanted := "the mit license (mit)\n\n\tsome other lines.\n\tand more.\n\t"
	if wanted != cleaned {
		t.Fatalf("license data mismatch:\n%q\n!=\n%q", cleaned, wanted)
	}
}

func TestStandardPackages(t *testing.T) {
	err := compareTestLicenses([]string{"encoding/json", "cmd/addr2line"}, []testResult{})
	if err != nil {
		t.Fatal(err)
	}
}

func TestOverrides(t *testing.T) {
	wl := []projectAndLicenses{
		{Project: "colors/broken", Licenses: []truncLicense{
			{Name: "GNU General Public License v3.0", Confidence: 1}},
		},
		{Project: "colors/red", Licenses: []truncLicense{
			{Name: "override existing", Confidence: 1}},
		},
		{Project: "colors/missing", Licenses: []truncLicense{
			{Name: "override missing", Confidence: 1}},
		},
	}
	override := `[
		{"project": "colors/missing", "licenses": [{"name": "override missing"}]},
		{"project": "colors/red", "licenses": [{"name": "override existing"}]}
	]`

	wd, derr := os.Getwd()
	if derr != nil {
		t.Fatal(derr)
	}
	oldenv := os.Getenv("GOPATH")
	defer os.Setenv("GOPATH", oldenv)
	os.Setenv("GOPATH", filepath.Join(wd, "testdata"))

	c, e := pkgsToLicenses([]string{"colors/broken"}, override)
	if len(e) != 0 {
		t.Fatalf("got %+v errors, expected nothing", e)
	}
	for i := range c {
		if !reflect.DeepEqual(wl[i], c[i]) {
			t.Errorf("#%d:\ngot      %+v,\nexpected %+v", i, c[i], wl[i])
		}
	}
}

func TestLongestPrefix(t *testing.T) {
	tests := []struct {
		lics []License

		wpfx string
	}{
		{
			[]License{
				{Package: "a/b/c"},
				{Package: "a/b/c/d"},
			},
			"a/b/c",
		},
		{
			[]License{
				{Package: "a/b/c"},
				{Package: "a/b/c/d"},
				{Package: "a/b/c/d/e"},
			},
			"a/b/c",
		},
		{
			[]License{
				{Package: "a/b"},
				{Package: "a/b/c/d/f"},
				{Package: "a/b/c/d/e"},
			},
			"a/b",
		},
	}

	for i, tt := range tests {
		if s := longestCommonPrefix(tt.lics); s != tt.wpfx {
			t.Errorf("#%d: got %q, expected %q", i, s, tt.wpfx)
		}
	}
}
