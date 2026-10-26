package arch_test

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

// Forbidden imports for domain packages. Business rules never see I/O.
var domainForbidden = []string{
	"net/http",
	"database/sql",
	"github.com/go-chi/",
	"github.com/jackc/pgx",
	"github.com/redis/go-redis",
	"github.com/golang-migrate/",
	"go.opentelemetry.io/",
	"github.com/aws/",
	"github.com/openai/",
	"treatment-plus-be/internal/platform/httpx",
	"treatment-plus-be/internal/platform/postgres",
	"treatment-plus-be/internal/platform/redisx",
	"treatment-plus-be/internal/bootstrap",
}

var contexts = []string{
	"identity", "clinic", "catalog", "plan", "media", "output", "share", "ai",
}

func TestDomainHasNoInfrastructureImports(t *testing.T) {
	for _, ctx := range contexts {
		imports := collectImports(t, filepath.Join("..", ctx, "domain"))
		for _, imp := range imports {
			for _, bad := range domainForbidden {
				if strings.Contains(imp, bad) {
					t.Errorf("domain %s imports %s", ctx, imp)
				}
			}
			if strings.Contains(imp, "/http") || strings.Contains(imp, "/pg") {
				t.Errorf("domain %s imports adapter %s", ctx, imp)
			}
		}
	}
}

func TestHTTPDoesNotImportPostgresAdapters(t *testing.T) {
	for _, ctx := range contexts {
		imports := collectImports(t, filepath.Join("..", ctx, "http"))
		for _, imp := range imports {
			if strings.Contains(imp, "/pg") || strings.Contains(imp, "github.com/jackc/pgx") {
				t.Errorf("%s/http imports persistence %s", ctx, imp)
			}
			if other := otherContextAdapter(ctx, imp); other != "" {
				t.Errorf("%s/http imports other context adapter %s", ctx, other)
			}
		}
	}
}

func TestPGDoesNotImportHTTPAdapters(t *testing.T) {
	for _, ctx := range contexts {
		imports := collectImports(t, filepath.Join("..", ctx, "pg"))
		for _, imp := range imports {
			if strings.HasSuffix(imp, "/http") || strings.Contains(imp, "github.com/go-chi/") {
				t.Errorf("%s/pg imports http %s", ctx, imp)
			}
			if other := otherContextAdapter(ctx, imp); other != "" {
				t.Errorf("%s/pg imports other context adapter %s", ctx, other)
			}
		}
	}
}

func TestAppDoesNotImportAdapters(t *testing.T) {
	for _, ctx := range contexts {
		imports := collectImports(t, filepath.Join("..", ctx, "app"))
		for _, imp := range imports {
			if strings.Contains(imp, "github.com/jackc/pgx") ||
				strings.Contains(imp, "github.com/go-chi/") ||
				strings.Contains(imp, "net/http") ||
				strings.Contains(imp, "github.com/redis/go-redis") {
				t.Errorf("%s/app imports infrastructure %s", ctx, imp)
			}
			if strings.Contains(imp, "/http") || strings.HasSuffix(imp, "/pg") {
				t.Errorf("%s/app imports adapter %s", ctx, imp)
			}
		}
	}
}

func otherContextAdapter(self, imp string) string {
	for _, ctx := range contexts {
		if ctx == self {
			continue
		}
		if strings.Contains(imp, "/"+ctx+"/http") || strings.Contains(imp, "/"+ctx+"/pg") {
			return imp
		}
	}
	return ""
}

func collectImports(t *testing.T, dir string) []string {
	t.Helper()
	if _, err := filepath.Abs(dir); err != nil {
		t.Fatal(err)
	}
	var out []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if osIsNotExist(err) {
				return nil
			}
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, spec := range f.Imports {
			out = append(out, strings.Trim(spec.Path.Value, `"`))
		}
		return nil
	})
	if err != nil && !osIsNotExist(err) {
		t.Fatal(err)
	}
	return out
}

func osIsNotExist(err error) bool {
	return err != nil && strings.Contains(err.Error(), "no such file or directory")
}
