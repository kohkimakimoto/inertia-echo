package v8go

import (
	"os"
	"testing"
)

func TestSsrEngine_Render(t *testing.T) {
	f := testTempJavaScriptFile(t, []byte(`
const a = "aaaa";
const isServer = typeof window === 'undefined';

print("isServer");
print(isServer);

`))
	e := NewSsrEngine()
	if err := e.LoadFile(f.Name()); err != nil {
		t.Fatal(err)
	}

	_, err := e.Render(nil)
	if err != nil {
		t.Fatal(err)
	}
}

func testTempJavaScriptFile(t *testing.T, content []byte) *os.File {
	t.Helper()
	tempFile, err := os.CreateTemp("", "*.js")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Remove(tempFile.Name()) })

	_, err = tempFile.Write(content)
	if err != nil {
		t.Fatal(err)
	}
	return tempFile
}
