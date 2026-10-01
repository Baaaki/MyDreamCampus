package pgtest

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpSection_KeepsOnlyTheUpStatements(t *testing.T) {
	up, err := upSection(`-- +goose Up
-- +goose StatementBegin
CREATE FUNCTION f() RETURNS int AS $$ SELECT 1 $$ LANGUAGE sql;
-- +goose StatementEnd
CREATE TABLE a (id int);

-- +goose Down
DROP TABLE a;
`)

	require.NoError(t, err)
	assert.Contains(t, up, "CREATE FUNCTION f()")
	assert.Contains(t, up, "CREATE TABLE a")
	assert.NotContains(t, up, "DROP TABLE")
	assert.NotContains(t, up, "+goose")
}

func TestUpSection_RefusesWhatItCannotRunFaithfully(t *testing.T) {
	_, err := upSection("-- +goose NO TRANSACTION\n-- +goose Up\nCREATE INDEX CONCURRENTLY i ON a (id);\n")
	assert.Error(t, err)

	_, err = upSection("CREATE TABLE a (id int);\n")
	assert.Error(t, err, "a file without an Up marker is not a goose migration")
}

func TestReadMigrations_ReturnsSQLFilesInNameOrder(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "002_b.sql"), []byte("-- +goose Up\nSELECT 2;\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "001_a.sql"), []byte("-- +goose Up\nSELECT 1;\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("not a migration"), 0o600))

	files, err := readMigrations(dir)

	require.NoError(t, err)
	require.Len(t, files, 2)
	assert.Equal(t, "001_a.sql", files[0].name)
	assert.Contains(t, files[0].sql, "SELECT 1")
	assert.Equal(t, "002_b.sql", files[1].name)
}

func TestReadMigrations_EmptyOrMissingDir_Errors(t *testing.T) {
	_, err := readMigrations(t.TempDir())
	assert.Error(t, err)

	_, err = readMigrations(filepath.Join(t.TempDir(), "missing"))
	assert.Error(t, err)
}
