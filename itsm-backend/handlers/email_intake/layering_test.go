package email_intake

import (
	"os"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHandlerDependenciesStayAtServiceBoundary(t *testing.T) {
	handlerType := reflect.TypeOf(Handler{})
	require.Equal(t, 2, handlerType.NumField())

	serviceField, ok := handlerType.FieldByName("svc")
	require.True(t, ok)
	require.Equal(t, reflect.TypeOf((*Service)(nil)), serviceField.Type)
	_, hasOnCallDependency := handlerType.FieldByName("onCall")
	require.False(t, hasOnCallDependency)

	source, err := os.ReadFile("handler.go")
	require.NoError(t, err)
	require.NotContains(t, string(source), "itsm-backend/ent")
}
