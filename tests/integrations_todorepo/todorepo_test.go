package integrations_todorepo

import (
	"context"
	"stg/internal/app/repository/todorepo"
	"strconv"
	"testing"

	steron "github.com/FluorescentTouch/testosteron/v2"
	"github.com/stretchr/testify/require"

	"stg/internal/app/domain"
)

func TestTodoRepo(t *testing.T) {
	cases := []func(*testing.T, *todorepo.Repository){
		testIntegrationTodoRepoAdd,
		testIntegrationTodoRepoAddAndList,
		testIntegrationTodoRepoListEmpty,
		testIntegrationTodoRepoListFilterByUser,
		testIntegrationTodoRepoDelete,
		testIntegrationTodoRepoDeleteWrongUser,
		testIntegrationTodoRepoToggle,
		testIntegrationTodoRepoToggleWrongUser,
	}

	repo, err := prepare(t)
	if err != nil {
		t.Fatal(err)
	}

	for i, tc := range cases {
		steron.SqlExecFromFile(t, steron.PgxClient(t).DB(), "TRUNCATE TABLE todos RESTART IDENTITY")
		t.Run("todoRepo."+strconv.Itoa(i), func(t *testing.T) {
			tc(t, repo)
		})
	}
}

func testIntegrationTodoRepoAdd(t *testing.T, repo *todorepo.Repository) {
	userID := nextUserID()

	id, err := repo.Add(context.Background(), domain.Todo{
		UserID:  userID,
		Message: "test todo",
	})
	require.NoError(t, err)
	require.Positive(t, id)
}

func testIntegrationTodoRepoAddAndList(t *testing.T, repo *todorepo.Repository) {
	userID := nextUserID()

	_, err := repo.Add(context.Background(), domain.Todo{
		UserID:  userID,
		Message: "first",
	})
	require.NoError(t, err)

	_, err = repo.Add(context.Background(), domain.Todo{
		UserID:  userID,
		Message: "second",
	})
	require.NoError(t, err)

	items, err := repo.List(context.Background(), userID)
	require.NoError(t, err)
	require.Len(t, items, 2)
}

func testIntegrationTodoRepoListEmpty(t *testing.T, repo *todorepo.Repository) {
	userID := nextUserID()

	items, err := repo.List(context.Background(), userID)
	require.NoError(t, err)
	require.Empty(t, items)
}

func testIntegrationTodoRepoListFilterByUser(t *testing.T, repo *todorepo.Repository) {
	user1 := nextUserID()
	user2 := nextUserID()

	_, err := repo.Add(context.Background(), domain.Todo{
		UserID:  user1,
		Message: "user1 todo",
	})
	require.NoError(t, err)

	_, err = repo.Add(context.Background(), domain.Todo{
		UserID:  user2,
		Message: "user2 todo",
	})
	require.NoError(t, err)

	items, err := repo.List(context.Background(), user1)
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, "user1 todo", items[0].Message)
}

func testIntegrationTodoRepoDelete(t *testing.T, repo *todorepo.Repository) {
	userID := nextUserID()

	id, err := repo.Add(context.Background(), domain.Todo{
		UserID:  userID,
		Message: "delete me",
	})
	require.NoError(t, err)

	err = repo.Delete(context.Background(), domain.Todo{
		ID:     id,
		UserID: userID,
	})
	require.NoError(t, err)

	items, err := repo.List(context.Background(), userID)
	require.NoError(t, err)
	require.Empty(t, items)
}

func testIntegrationTodoRepoDeleteWrongUser(t *testing.T, repo *todorepo.Repository) {
	user1 := nextUserID()
	user2 := nextUserID()

	id, err := repo.Add(context.Background(), domain.Todo{
		UserID:  user1,
		Message: "owned by user1",
	})
	require.NoError(t, err)

	err = repo.Delete(context.Background(), domain.Todo{
		ID:     id,
		UserID: user2,
	})
	require.NoError(t, err)

	items, err := repo.List(context.Background(), user1)
	require.NoError(t, err)
	require.Len(t, items, 1)
}

func testIntegrationTodoRepoToggle(t *testing.T, repo *todorepo.Repository) {
	userID := nextUserID()

	id, err := repo.Add(context.Background(), domain.Todo{
		UserID:  userID,
		Message: "toggle me",
	})
	require.NoError(t, err)

	err = repo.Toggle(context.Background(), domain.Todo{
		ID:     id,
		UserID: userID,
		Done:   true,
	})
	require.NoError(t, err)

	items, err := repo.List(context.Background(), userID)
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.True(t, items[0].Done)
}

func testIntegrationTodoRepoToggleWrongUser(t *testing.T, repo *todorepo.Repository) {
	user1 := nextUserID()
	user2 := nextUserID()

	id, err := repo.Add(context.Background(), domain.Todo{
		UserID:  user1,
		Message: "owned by user1",
	})
	require.NoError(t, err)

	err = repo.Toggle(context.Background(), domain.Todo{
		ID:     id,
		UserID: user2,
		Done:   true,
	})
	require.NoError(t, err)

	items, err := repo.List(context.Background(), user1)
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.False(t, items[0].Done)
}
