package main

import (
	"encoding/json"
	"math/rand"
	"reflect"
	"testing"
	"testing/quick"
	"time"
)

// **Feature: video-generation-ui, Property 6: Task JSON serialization round-trip**
// **Validates: Requirements 6.2, 6.3, 6.4**
//
// For any valid Task struct, serializing to JSON and then deserializing back
// should produce an equivalent Task struct with all fields preserved.
func TestTaskJSONSerializationRoundTrip(t *testing.T) {
	config := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	property := func(task Task) bool {
		// Serialize to JSON
		jsonBytes, err := json.Marshal(task)
		if err != nil {
			t.Logf("Failed to marshal task: %v", err)
			return false
		}

		// Deserialize back to Task
		var deserializedTask Task
		err = json.Unmarshal(jsonBytes, &deserializedTask)
		if err != nil {
			t.Logf("Failed to unmarshal task: %v", err)
			return false
		}

		// Compare all fields
		if !reflect.DeepEqual(task, deserializedTask) {
			t.Logf("Round-trip failed:\nOriginal: %+v\nDeserialized: %+v", task, deserializedTask)
			return false
		}

		return true
	}

	if err := quick.Check(property, config); err != nil {
		t.Errorf("Property test failed: %v", err)
	}
}

// Generate implements quick.Generator for Task
func (Task) Generate(rand *rand.Rand, size int) reflect.Value {
	statuses := []string{StatusPending, StatusProcessing, StatusCompleted, StatusFailed}
	durations := []string{Duration10s, Duration15s}
	orientations := []string{OrientationPortrait, OrientationLandscape}

	task := Task{
		ID:          rand.Int63(),
		TaskID:      randomString(rand, 10),
		Prompt:      randomString(rand, 50),
		ImageURL:    randomOptionalString(rand, 100),
		Duration:    durations[rand.Intn(len(durations))],
		Orientation: orientations[rand.Intn(len(orientations))],
		Status:      statuses[rand.Intn(len(statuses))],
		Progress:    rand.Intn(101), // 0-100
		VideoURL:    randomOptionalString(rand, 100),
		LocalPath:   randomOptionalString(rand, 50),
		CreatedAt:   randomTime(rand),
		UpdatedAt:   randomTime(rand),
	}

	return reflect.ValueOf(task)
}

// Helper function to generate random strings
func randomString(rand *rand.Rand, maxLen int) string {
	length := rand.Intn(maxLen) + 1
	chars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = chars[rand.Intn(len(chars))]
	}
	return string(result)
}

// Helper function to generate optional strings (empty or random)
func randomOptionalString(rand *rand.Rand, maxLen int) string {
	if rand.Intn(2) == 0 {
		return ""
	}
	return randomString(rand, maxLen)
}

// Helper function to generate random time
func randomTime(rand *rand.Rand) time.Time {
	// Generate time between 2020-01-01 and 2030-01-01
	min := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC).Unix()
	max := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC).Unix()
	sec := rand.Int63n(max-min) + min
	return time.Unix(sec, 0).UTC()
}
