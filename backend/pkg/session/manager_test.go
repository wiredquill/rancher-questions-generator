package session

import (
	"sync"
	"testing"
	"time"

	"rancher-questions-generator/internal/models"
)

func TestNewManager(t *testing.T) {
	manager := NewManager()
	if manager == nil {
		t.Fatal("NewManager() returned nil")
	}
	
	if manager.sessions == nil {
		t.Error("sessions map not initialized")
	}
}

func TestCreateSession(t *testing.T) {
	manager := NewManager()
	chartURL := "https://charts.example.com/chart.tgz"
	
	session := manager.CreateSession(chartURL)
	
	if session == nil {
		t.Fatal("CreateSession() returned nil")
	}
	
	if session.ID == "" {
		t.Error("Session ID is empty")
	}
	
	if session.ChartURL != chartURL {
		t.Errorf("Expected ChartURL %s, got %s", chartURL, session.ChartURL)
	}
	
	if session.Values == nil {
		t.Error("Values map not initialized")
	}
	
	if session.Questions.Questions == nil {
		t.Error("Questions slice not initialized")
	}
	
	if session.CreatedAt.IsZero() {
		t.Error("CreatedAt not set")
	}
	
	if session.UpdatedAt.IsZero() {
		t.Error("UpdatedAt not set")
	}
	
	// Verify session is stored in manager
	storedSession, err := manager.GetSession(session.ID)
	if err != nil {
		t.Errorf("Failed to retrieve created session: %v", err)
	}
	
	if storedSession.ID != session.ID {
		t.Error("Stored session ID mismatch")
	}
}

func TestGetSession(t *testing.T) {
	manager := NewManager()
	
	// Test non-existent session
	_, err := manager.GetSession("non-existent")
	if err == nil {
		t.Error("Expected error for non-existent session")
	}
	
	// Create a session and retrieve it
	chartURL := "https://charts.example.com/chart.tgz"
	createdSession := manager.CreateSession(chartURL)
	
	retrievedSession, err := manager.GetSession(createdSession.ID)
	if err != nil {
		t.Errorf("Failed to get existing session: %v", err)
	}
	
	if retrievedSession.ID != createdSession.ID {
		t.Error("Retrieved session ID mismatch")
	}
	
	if retrievedSession.ChartURL != chartURL {
		t.Error("Retrieved session ChartURL mismatch")
	}
}

func TestUpdateSession(t *testing.T) {
	manager := NewManager()
	
	// Test updating non-existent session
	questions := models.Questions{
		Questions: []models.Question{
			{
				Variable: "test.var",
				Label:    "Test Variable",
				Type:     "string",
			},
		},
	}
	
	err := manager.UpdateSession("non-existent", questions)
	if err == nil {
		t.Error("Expected error for non-existent session")
	}
	
	// Create a session and update it
	session := manager.CreateSession("https://charts.example.com/chart.tgz")
	originalUpdateTime := session.UpdatedAt
	
	// Wait a bit to ensure timestamp difference
	time.Sleep(10 * time.Millisecond)
	
	err = manager.UpdateSession(session.ID, questions)
	if err != nil {
		t.Errorf("Failed to update session: %v", err)
	}
	
	// Verify the update
	updatedSession, err := manager.GetSession(session.ID)
	if err != nil {
		t.Errorf("Failed to get updated session: %v", err)
	}
	
	if len(updatedSession.Questions.Questions) != 1 {
		t.Errorf("Expected 1 question, got %d", len(updatedSession.Questions.Questions))
	}
	
	if updatedSession.Questions.Questions[0].Variable != "test.var" {
		t.Error("Question variable not updated correctly")
	}
	
	if updatedSession.UpdatedAt.Before(originalUpdateTime) || updatedSession.UpdatedAt.Equal(originalUpdateTime) {
		t.Error("UpdatedAt timestamp not updated")
	}
}

func TestDeleteSession(t *testing.T) {
	manager := NewManager()
	
	// Test deleting non-existent session
	err := manager.DeleteSession("non-existent")
	if err == nil {
		t.Error("Expected error for non-existent session")
	}
	
	// Create a session and delete it
	session := manager.CreateSession("https://charts.example.com/chart.tgz")
	
	err = manager.DeleteSession(session.ID)
	if err != nil {
		t.Errorf("Failed to delete session: %v", err)
	}
	
	// Verify deletion
	_, err = manager.GetSession(session.ID)
	if err == nil {
		t.Error("Session still exists after deletion")
	}
}

func TestConcurrentAccess(t *testing.T) {
	manager := NewManager()
	numGoroutines := 100
	
	var wg sync.WaitGroup
	sessionIDs := make([]string, numGoroutines)
	
	// Concurrent session creation
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(index int) {
			defer wg.Done()
			chartURL := fmt.Sprintf("https://charts.example%d.com/chart.tgz", index)
			session := manager.CreateSession(chartURL)
			sessionIDs[index] = session.ID
		}(i)
	}
	wg.Wait()
	
	// Verify all sessions were created
	for i, sessionID := range sessionIDs {
		if sessionID == "" {
			t.Errorf("Session %d was not created", i)
			continue
		}
		
		session, err := manager.GetSession(sessionID)
		if err != nil {
			t.Errorf("Failed to retrieve session %d: %v", i, err)
			continue
		}
		
		expectedURL := fmt.Sprintf("https://charts.example%d.com/chart.tgz", i)
		if session.ChartURL != expectedURL {
			t.Errorf("Session %d URL mismatch: expected %s, got %s", i, expectedURL, session.ChartURL)
		}
	}
	
	// Concurrent updates
	questions := models.Questions{
		Questions: []models.Question{
			{
				Variable: "concurrent.test",
				Label:    "Concurrent Test",
				Type:     "boolean",
			},
		},
	}
	
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(index int) {
			defer wg.Done()
			if sessionIDs[index] != "" {
				manager.UpdateSession(sessionIDs[index], questions)
			}
		}(i)
	}
	wg.Wait()
	
	// Concurrent reads
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(index int) {
			defer wg.Done()
			if sessionIDs[index] != "" {
				manager.GetSession(sessionIDs[index])
			}
		}(i)
	}
	wg.Wait()
	
	// Concurrent deletions
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(index int) {
			defer wg.Done()
			if sessionIDs[index] != "" {
				manager.DeleteSession(sessionIDs[index])
			}
		}(i)
	}
	wg.Wait()
	
	// Verify all sessions were deleted
	for i, sessionID := range sessionIDs {
		if sessionID == "" {
			continue
		}
		
		_, err := manager.GetSession(sessionID)
		if err == nil {
			t.Errorf("Session %d still exists after deletion", i)
		}
	}
}

func TestSessionDataIntegrity(t *testing.T) {
	manager := NewManager()
	
	chartURL := "https://charts.example.com/complex-chart.tgz"
	session := manager.CreateSession(chartURL)
	
	// Add complex data
	complexValues := map[string]interface{}{
		"simple":  "value",
		"number":  42,
		"boolean": true,
		"nested": map[string]interface{}{
			"deep": map[string]interface{}{
				"value": "nested-value",
			},
		},
		"array": []interface{}{1, 2, 3},
	}
	
	session.Values = complexValues
	
	complexQuestions := models.Questions{
		Questions: []models.Question{
			{
				Variable:    "app.name",
				Label:       "Application Name",
				Description: "Name of the application",
				Type:        "string",
				Required:    true,
				Default:     "my-app",
				Group:       "General",
				Options:     []string{"option1", "option2"},
				ShowIf:      "advanced=true",
				SubQuestions: []models.Question{
					{
						Variable: "app.subconfig",
						Label:    "Sub Configuration",
						Type:     "boolean",
					},
				},
			},
		},
	}
	
	err := manager.UpdateSession(session.ID, complexQuestions)
	if err != nil {
		t.Fatalf("Failed to update session with complex data: %v", err)
	}
	
	// Retrieve and verify data integrity
	retrievedSession, err := manager.GetSession(session.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve session: %v", err)
	}
	
	// Verify values
	if retrievedSession.Values["simple"] != "value" {
		t.Error("Simple value integrity lost")
	}
	
	if retrievedSession.Values["number"] != 42 {
		t.Error("Number value integrity lost")
	}
	
	if retrievedSession.Values["boolean"] != true {
		t.Error("Boolean value integrity lost")
	}
	
	nested, ok := retrievedSession.Values["nested"].(map[string]interface{})
	if !ok {
		t.Error("Nested value type integrity lost")
	} else {
		deep, ok := nested["deep"].(map[string]interface{})
		if !ok {
			t.Error("Deep nested value type integrity lost")
		} else if deep["value"] != "nested-value" {
			t.Error("Deep nested value integrity lost")
		}
	}
	
	// Verify questions
	if len(retrievedSession.Questions.Questions) != 1 {
		t.Errorf("Expected 1 question, got %d", len(retrievedSession.Questions.Questions))
	}
	
	q := retrievedSession.Questions.Questions[0]
	if q.Variable != "app.name" {
		t.Error("Question variable integrity lost")
	}
	
	if q.Label != "Application Name" {
		t.Error("Question label integrity lost")
	}
	
	if !q.Required {
		t.Error("Question required flag integrity lost")
	}
	
	if q.Default != "my-app" {
		t.Error("Question default value integrity lost")
	}
	
	if len(q.Options) != 2 {
		t.Error("Question options integrity lost")
	}
	
	if len(q.SubQuestions) != 1 {
		t.Error("Question subquestions integrity lost")
	}
}

func TestSessionTimestamps(t *testing.T) {
	manager := NewManager()
	
	beforeCreate := time.Now()
	session := manager.CreateSession("https://charts.example.com/chart.tgz")
	afterCreate := time.Now()
	
	// Verify creation timestamps
	if session.CreatedAt.Before(beforeCreate) || session.CreatedAt.After(afterCreate) {
		t.Error("CreatedAt timestamp is outside expected range")
	}
	
	if session.UpdatedAt.Before(beforeCreate) || session.UpdatedAt.After(afterCreate) {
		t.Error("Initial UpdatedAt timestamp is outside expected range")
	}
	
	if !session.CreatedAt.Equal(session.UpdatedAt) {
		t.Error("CreatedAt and UpdatedAt should be equal on creation")
	}
	
	// Wait and update
	time.Sleep(10 * time.Millisecond)
	beforeUpdate := time.Now()
	
	questions := models.Questions{
		Questions: []models.Question{
			{Variable: "test", Label: "Test", Type: "string"},
		},
	}
	
	err := manager.UpdateSession(session.ID, questions)
	if err != nil {
		t.Fatalf("Failed to update session: %v", err)
	}
	
	afterUpdate := time.Now()
	
	// Verify update timestamps
	updatedSession, _ := manager.GetSession(session.ID)
	
	if updatedSession.CreatedAt != session.CreatedAt {
		t.Error("CreatedAt should not change on update")
	}
	
	if updatedSession.UpdatedAt.Before(beforeUpdate) || updatedSession.UpdatedAt.After(afterUpdate) {
		t.Error("UpdatedAt timestamp is outside expected range after update")
	}
	
	if !updatedSession.UpdatedAt.After(session.UpdatedAt) {
		t.Error("UpdatedAt should be after original timestamp")
	}
}

// Benchmark tests
func BenchmarkCreateSession(b *testing.B) {
	manager := NewManager()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		chartURL := fmt.Sprintf("https://charts.example%d.com/chart.tgz", i)
		manager.CreateSession(chartURL)
	}
}

func BenchmarkGetSession(b *testing.B) {
	manager := NewManager()
	session := manager.CreateSession("https://charts.example.com/chart.tgz")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager.GetSession(session.ID)
	}
}

func BenchmarkUpdateSession(b *testing.B) {
	manager := NewManager()
	session := manager.CreateSession("https://charts.example.com/chart.tgz")
	
	questions := models.Questions{
		Questions: []models.Question{
			{Variable: "test", Label: "Test", Type: "string"},
		},
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager.UpdateSession(session.ID, questions)
	}
}

func BenchmarkConcurrentOperations(b *testing.B) {
	manager := NewManager()
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// Mix of operations
			session := manager.CreateSession("https://charts.example.com/chart.tgz")
			manager.GetSession(session.ID)
			
			questions := models.Questions{
				Questions: []models.Question{
					{Variable: "test", Label: "Test", Type: "string"},
				},
			}
			manager.UpdateSession(session.ID, questions)
			manager.DeleteSession(session.ID)
		}
	})
}