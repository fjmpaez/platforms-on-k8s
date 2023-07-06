package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	tc "github.com/testcontainers/testcontainers-go/modules/compose"
	"github.com/testcontainers/testcontainers-go/wait"
)

// testServer returns a httptest.Server for testing.
func testServer() *httptest.Server {
	chiServer := NewChiServer()
	return httptest.NewServer(chiServer)
}

func Test_API(t *testing.T) {

	// testcontainers
	compose, err := tc.NewDockerCompose("docker-compose.yaml")
	assert.NoError(t, err, "NewDockerComposeAPI()")

	t.Cleanup(func() {
		assert.NoError(t, compose.Down(context.Background()), tc.RemoveOrphans(true))
	})

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	err = compose.
		WaitForService("kafka", wait.ForListeningPort("9094")).
		Up(ctx, tc.Wait(true))

	assert.NoError(t, err, "compose.Up()")

	// test server
	ts := testServer()
	defer ts.Close()

	t.Run("It should return 200 when a GET request is made to '/health/readiness'", func(t *testing.T) {
		// arrange, act
		res, _ := http.Get(fmt.Sprintf("%s/health/readiness", ts.URL))

		// assert
		assert.Equal(t, http.StatusOK, res.StatusCode)
	})

	t.Run("It should return 200 when a GET request is made to '/health/liveness'", func(t *testing.T) {
		// arrange, act
		resp, _ := http.Get(fmt.Sprintf("%s/health/liveness", ts.URL))

		// assert
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("It should return 200 when a GET request is made to '/service/info'", func(t *testing.T) {
		// arrange, act
		resp, _ := http.Get(fmt.Sprintf("%s/service/info", ts.URL))

		// assert
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("It should return 200 when a POST request is made to '/notifications' (accepted)", func(t *testing.T) {
		// arrange
		var accepted bool = true
		notification := notificationFake(accepted)

		notificationAsBytes, _ := notification.MarshalBinary()

		// act
		resp, _ := http.Post(fmt.Sprintf("%s/notifications", ts.URL), "application/json", bytes.NewBuffer(notificationAsBytes))

		// assert
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("It should return 200 when a POST request is made to '/notifications' (not accepted)", func(t *testing.T) {
		// arrange
		var accepted bool = false
		notification := notificationFake(accepted)

		notificationAsBytes, _ := notification.MarshalBinary()

		// act
		resp, _ := http.Post(fmt.Sprintf("%s/notifications", ts.URL), "application/json", bytes.NewBuffer(notificationAsBytes))

		// assert
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("It should return 200 when a GET request is made to '/notifications'", func(t *testing.T) {
		// arrange, act
		resp, err := http.Get(fmt.Sprintf("%s/notifications", ts.URL))

		defer resp.Body.Close()

		var notifications []Notification
		json.NewDecoder(resp.Body).Decode(&notifications)

		// assert
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.True(t, len(notifications) > 0)
	})

}

func notificationFake(accepted bool) Notification {
	return Notification{
		ProposalId: uuid.New().String(),
		Title:      "Dapr + Crossplane",
		Accepted:   accepted,
		EmailTo:    "salaboy@salaboy.com",
	}
}