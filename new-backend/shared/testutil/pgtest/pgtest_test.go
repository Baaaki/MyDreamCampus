package pgtest

import (
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
