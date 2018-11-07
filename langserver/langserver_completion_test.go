package langserver

import (
	"context"
	"fmt"
	"github.com/saibing/bingo/pkg/lsp"
	"github.com/sourcegraph/jsonrpc2"
	"log"
	"path/filepath"
	"strings"
	"testing"

	"github.com/saibing/bingo/langserver/util"
)

func TestCompletion(t *testing.T) {
	test := func(t *testing.T, pkgDir string, input string, output string) {
		testCompletion(t, &completionTestCase{pkgDir: pkgDir, input: input, output: output})
	}

	t.Run("basic", func(t *testing.T) {
		test(t, basicPkgDir, "b.go:1:24", "1:23-1:24 A function func()")
	})

	t.Run("xtest", func(t *testing.T) {
		test(t, xtestPkgDir, "x_test.go:1:45", "1:44-1:45 panic function func(interface{}), print function func(...interface{}), println function func(...interface{}), p module ")
		test(t, xtestPkgDir, "x_test.go:1:46", "1:46-1:46 A variable int")
		test(t, xtestPkgDir, "b_test.go:1:35", "1:34-1:35 X variable int")
	})

	t.Run("go subdirectory in repo", func(t *testing.T) {
		test(t, subdirectoryPkgDir, "d2/b.go:1:94", "1:94-1:94 A function func()")
	})

	t.Run("go root", func(t *testing.T) {
		test(t, gorootPkgDir, "a.go:1:21", "1:20-1:21 flag module , fmt module ")
		test(t, gorootPkgDir, "a.go:1:44", "1:38-1:44 Println function func(a ...interface{}) (n int, err error)")
	})

	t.Run("go project workspace", func(t *testing.T) {
		test(t, goprojectPkgDir, "b/b.go:1:26", "1:20-1:26 test/pkg/a module , test/pkg/b module ")
		test(t, goprojectPkgDir, "b/b.go:1:43", "1:43-1:43 A function func()")
	})

	t.Run("go module dep", func(t *testing.T) {
		test(t, gomodulePkgDir, "a.go:1:40", "1:20-1:40 github.com/saibing/dep module ")
		test(t, gomodulePkgDir, "a.go:1:57", "1:57-1:57 D function func()")

		test(t, gomodulePkgDir, "b.go:1:40", "1:20-1:40 github.com/saibing/dep/subp module ")
		test(t, gomodulePkgDir, "b.go:1:63", "1:57-1:57 D function func()")

		test(t, gomodulePkgDir, "c.go:1:68", "1:58-1:58 D2 variable int")
	})

	t.Run("complete", func(t *testing.T) {
		/*test(t, completionPkgDir, "a.go:6:7", "6:6-6:7 s1 constant , s2 function func(), strings module , string class built-in, s3 variable int, s4 variable func()")
		test(t, completionPkgDir, "a.go:7:7", "7:6-7:7 nil constant , new function func(type) *type")
		test(t, completionPkgDir, "a.go:12:11", "12:8-12:11 int class built-in, int16 class built-in, int32 class built-in, int64 class built-in, int8 class built-in")
		test(t, completionPkgDir, "b.go:1:44", "1:38-1:44 Println function func(a ...interface{}) (n int, err error)")*/
		//test(t, completionPkgDir, "c.go:1:38", "1:34-1:38 Errorf function func(format string, a ...interface{}) error, Formatter  type fmt.Formatter interface{Format(f fmt.State, c rune)}, Fprint function func(w io.Writer, a ...interface{}) (n int, err error), Fprintf function func(w io.Writer, format string, a ...interface{}) (n int, err error), Fprintln function func(w io.Writer, a ...interface{}) (n int, err error), Fscan function func(r io.Reader, a ...interface{}) (n int, err error), Fscanf function func(r io.Reader, format string, a ...interface{}) (n int, err error), Fscanln function func(r io.Reader, a ...interface{}) (n int, err error), GoStringer  type fmt.GoStringer interface{GoString() string}, Print function func(a ...interface{}) (n int, err error), Printf function func(format string, a ...interface{}) (n int, err error), Println function func(a ...interface{}) (n int, err error), Scan function func(a ...interface{}) (n int, err error), ScanState  type fmt.ScanState interface{Read(buf []byte) (n int, err error); ReadRune() (r rune, size int, err error); SkipSpace(); Token(skipSpace bool, f func(rune) bool) (token []byte, err error); UnreadRune() error; Width() (wid int, ok bool)}, Scanf function func(format string, a ...interface{}) (n int, err error), Scanln function func(a ...interface{}) (n int, err error), Scanner  type fmt.Scanner interface{Scan(state fmt.ScanState, verb rune) error}, Sprint function func(a ...interface{}) string, Sprintf function func(format string, a ...interface{}) string, Sprintln function func(a ...interface{}) string, Sscan function func(str string, a ...interface{}) (n int, err error), Sscanf function func(str string, format string, a ...interface{}) (n int, err error), Sscanln function func(str string, a ...interface{}) (n int, err error), State  type fmt.State interface{Flag(c int) bool; Precision() (prec int, ok bool); Width() (wid int, ok bool); Write(b []byte) (n int, err error)}, Stringer  type fmt.Stringer interface{String() string}")

		test(t, completionPkgDir, "d.go:14:5", "1:38-1:44 Println function func(a ...interface{}) (n int, err error)")
	})
}

type completionTestCase struct {
	pkgDir string
	input  string
	output string
}

func testCompletion(tb testing.TB, c *completionTestCase) {
	tbRun(tb, fmt.Sprintf("complete-%s", strings.Replace(c.input, "/", "-", -1)), func(t testing.TB) {
		dir, err := filepath.Abs(c.pkgDir)
		if err != nil {
			log.Fatal("testCompletion", err)
		}
		doCompletionTest(t, ctx, conn, util.PathToURI(dir), c.input, c.output)
	})
}

func doCompletionTest(t testing.TB, ctx context.Context, c *jsonrpc2.Conn, rootURI lsp.DocumentURI, pos, want string) {
	file, line, char, err := parsePos(pos)
	if err != nil {
		t.Fatal(err)
	}
	completion, err := callCompletion(ctx, c, uriJoin(rootURI, file), line, char)
	if err != nil {
		t.Fatal(err)
	}
	if completion != want {
		t.Fatalf("got %q, want %q", completion, want)
	}
}

func callCompletion(ctx context.Context, c *jsonrpc2.Conn, uri lsp.DocumentURI, line, char int) (string, error) {
	var res lsp.CompletionList
	err := c.Call(ctx, "textDocument/completion", lsp.CompletionParams{TextDocumentPositionParams: lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{URI: uri},
		Position:     lsp.Position{Line: line, Character: char},
	}}, &res)
	if err != nil {
		return "", err
	}
	var str string
	for i, it := range res.Items {
		if i != 0 {
			str += ", "
		} else {
			e := it.TextEdit.Range
			str += fmt.Sprintf("%d:%d-%d:%d ", e.Start.Line+1, e.Start.Character+1, e.End.Line+1, e.End.Character+1)
		}
		str += fmt.Sprintf("%s %s %s", it.Label, it.Kind, it.Detail)
	}
	return str, nil
}