package e2e_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"stg/internal/app/router/todolistctrl"
)

func TestE2ETodo_Create(t *testing.T) {
	userID := nextUserID()
	authenticateUser(t, userID)

	createResp := postJSON(t, "/v1/todos/create", todolistctrl.CreateRequest{
		Message: "e2e todo",
	})
	requireStatus(t, http.StatusOK, createResp)
	defer createResp.Body.Close()
	created := decodeJSON[todolistctrl.CreateResponse](t, createResp)
	require.Positive(t, created.ID)

	listResp := postJSON(t, "/v1/todos/list", nil)
	requireStatus(t, http.StatusOK, listResp)
	defer listResp.Body.Close()
	items := decodeJSON[[]todolistctrl.TodoItem](t, listResp)
	require.Len(t, items, 1)

	got := todoItemToDomain(t, items[0], userID)
	require.Equal(t, created.ID, got.ID)
	require.Equal(t, "e2e todo", got.Message)
	require.False(t, got.Done)
}

func TestE2ETodo_List(t *testing.T) {
	userID := nextUserID()
	authenticateUser(t, userID)

	createTodo(t, userID, "todo one")
	createTodo(t, userID, "todo two")

	resp := postJSON(t, "/v1/todos/list", nil)
	requireStatus(t, http.StatusOK, resp)
	defer resp.Body.Close()
	items := decodeJSON[[]todolistctrl.TodoItem](t, resp)
	require.Len(t, items, 2)
}

func TestE2ETodo_Toggle(t *testing.T) {
	userID := nextUserID()
	authenticateUser(t, userID)
	todoID := createTodo(t, userID, "toggle me")

	toggleResp := postJSON(t, "/v1/todos/toggle", todolistctrl.ToggleRequest{
		ID: todoID, Done: true,
	})
	requireStatus(t, http.StatusOK, toggleResp)
	defer toggleResp.Body.Close()

	listResp := postJSON(t, "/v1/todos/list", nil)
	requireStatus(t, http.StatusOK, listResp)
	defer listResp.Body.Close()
	items := decodeJSON[[]todolistctrl.TodoItem](t, listResp)
	require.NotEmpty(t, items)
	require.True(t, todoItemToDomain(t, items[0], userID).Done)
}

func TestE2ETodo_Delete(t *testing.T) {
	userID := nextUserID()
	authenticateUser(t, userID)
	todoID := createTodo(t, userID, "delete me")

	deleteResp := postJSON(t, "/v1/todos/delete", todolistctrl.DeleteRequest{
		ID: todoID,
	})
	requireStatus(t, http.StatusOK, deleteResp)
	defer deleteResp.Body.Close()

	listResp := postJSON(t, "/v1/todos/list", nil)
	requireStatus(t, http.StatusOK, listResp)
	defer listResp.Body.Close()
	items := decodeJSON[[]todolistctrl.TodoItem](t, listResp)
	require.Empty(t, items)
}

func TestE2ETodo_List_Unauthorized(t *testing.T) {
	// Без аутентификации — куки нет, ожидаем 401
	// Используем DefaultClient без CookieJar
	req, err := http.NewRequest(http.MethodPost, stgURL("/v1/todos/list"), http.NoBody)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestE2ETodo_Create_Unauthorized(t *testing.T) {
	req, err := http.NewRequest(http.MethodPost, stgURL("/v1/todos/create"), http.NoBody)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// --- Helpers ---

func createTodo(t *testing.T, userID int64, message string) int64 {
	t.Helper()

	resp := postJSON(t, "/v1/todos/create", todolistctrl.CreateRequest{
		Message: message,
	})
	requireStatus(t, http.StatusOK, resp)
	defer resp.Body.Close()

	created := decodeJSON[todolistctrl.CreateResponse](t, resp)
	require.Positive(t, created.ID)

	return created.ID
}
