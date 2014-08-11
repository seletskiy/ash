package main

import (
	"bytes"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"testing"
)

type diffTest struct {
	in  []byte
	out []byte
	err []byte
}

func TestParseDiff(t *testing.T) {
	for name, testCase := range getTests("_test") {
		expected := string(testCase.out)

		diffs, err := ParseDiff(bytes.NewBuffer(testCase.in))
		if err != nil {
			t.Fatal(err)
		}

		actual := diffs.String()
		if actual != expected {
			t.Logf("while testing on `%s`\n", name)
			t.Logf("expected:\n%v", expected)
			t.Logf("actual:\n%v", actual)
			t.Logf("diff:\n%v", getDiff(actual, expected))
			t.FailNow()
		}
	}
}

func getTests(dir string) map[string]*diffTest {
	diffTests := make(map[string]*diffTest)
	reDiffTest := regexp.MustCompile(`/([^/]+)\.(in|out|err)\.diff$`)

	filepath.Walk(dir, func(path string, f os.FileInfo, err error) error {
		if !reDiffTest.MatchString(path) {
			return nil
		}

		matches := reDiffTest.FindStringSubmatch(path)

		caseName := matches[1]

		if _, exist := diffTests[caseName]; !exist {
			diffTests[caseName] = &diffTest{}
		}

		var target *[]byte

		switch matches[2] {
		case "in":
			target = &diffTests[caseName].in
		case "out":
			target = &diffTests[caseName].out
		case "err":
			target = &diffTests[caseName].err
		}

		*target, _ = ioutil.ReadFile(path)

		return nil
	})

	for _, val := range diffTests {
		if val.out == nil {
			val.out = val.in
		}
	}

	return diffTests
}

func getDiff(actual, expected string) string {
	a, _ := ioutil.TempFile(os.TempDir(), "actual")
	defer func() {
		os.Remove(a.Name())
	}()
	b, _ := ioutil.TempFile(os.TempDir(), "expected")
	defer func() {
		os.Remove(b.Name())
	}()

	a.WriteString(actual)
	b.WriteString(expected)
	cmd := exec.Command("diff", "-u", b.Name(), a.Name())
	buf := bytes.NewBuffer([]byte{})
	cmd.Stdout = buf
	cmd.Run()

	return buf.String()
}
