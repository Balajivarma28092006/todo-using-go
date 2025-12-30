package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/Balajivarma28092006/todo-using-go/internal/models"
	"github.com/Balajivarma28092006/todo-using-go/internal/storage"
	tea "github.com/charmbracelet/bubbletea"
)

type mockStorage struct{}

func (m *mockStorage) SaveTodo(todo *models.Todo) error {
	return nil
}

func (m *mockStorage) GetTodo(id string) (*models.Todo, error) {
	return nil, nil
}

func (m *mockStorage) GetAllTodos() ([]*models.Todo, error) {
	return []*models.Todo{}, nil
}

func (m *mockStorage) UpdateTodo(todo *models.Todo) error {
	return nil
}

func (m *mockStorage) DeleteTodo(id string) error {
	return nil
}

func (m *mockStorage) GetStreak() (*storage.Streak, error) {
	return &storage.Streak{
		CurrentStreak:    0,
		MaxStreak:        0,
		TotalCompleted:   0,
		DailyCompletions: make(map[string]int),
		LastCompletedAt:  time.Time{},
	}, nil
}

func (m *mockStorage) UpdateStreak(streak *storage.Streak) error {
	return nil
}

func (m *mockStorage) Close() error {
	return nil
}

func TestFormModel_CharacterLimit(t *testing.T) {
	tests := []struct {
		name         string
		fieldType    formField
		inputLength  int
		expectAccept bool
		maxLength    int
	}{
		{
			name:         "Title within limit",
			fieldType:    titleField,
			inputLength:  50,
			expectAccept: true,
			maxLength:    MaxTitleLenght,
		},
		{
			name:         "Title at exact limit",
			fieldType:    titleField,
			inputLength:  MaxTitleLenght,
			expectAccept: true,
			maxLength:    MaxTitleLenght,
		},
		{
			name:         "Title exceeds limit",
			fieldType:    titleField,
			inputLength:  MaxTitleLenght + 1,
			expectAccept: false,
			maxLength:    MaxTitleLenght,
		},
		{
			name:         "Description within limit",
			fieldType:    descriptionField,
			inputLength:  250,
			expectAccept: true,
			maxLength:    MaxDescriptionLenght,
		},
		{
			name:         "Description at exact limit",
			fieldType:    descriptionField,
			inputLength:  MaxDescriptionLenght,
			expectAccept: true,
			maxLength:    MaxDescriptionLenght,
		},
		{
			name:         "Description exceeds limit",
			fieldType:    descriptionField,
			inputLength:  MaxDescriptionLenght + 1,
			expectAccept: false,
			maxLength:    MaxDescriptionLenght,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := &mockStorage{}
			model := NewFormModel(mockStore)
			model.currentField = tt.fieldType

			// Try to add characters upto the input length
			for i := 0; i < tt.inputLength; i++ {
				msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}
				updateModel, _ := model.Update(msg)
				model = updateModel.(*FormModel)
			}

			actualLength := len(model.fields[tt.fieldType])

			if tt.expectAccept {
				if actualLength != tt.inputLength {
					t.Errorf("Expected field length %d, got %d", tt.inputLength, actualLength)
				}
			} else {
				if actualLength != tt.maxLength {
					t.Errorf("Expected field to be limited to %d characters, got %d", tt.maxLength, actualLength)
				}
			}
		})
	}
}

func TestFormModel_ValidateSubmission(t *testing.T) {
	tests := []struct {
		name        string
		title       string
		description string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "Empty title",
			title:       "",
			description: "Test Description",
			expectError: true,
			errorMsg:    "title is required",
		},
		{
			name:        "Empty description",
			title:       "Test Todo",
			description: "",
			expectError: true,
			errorMsg:    "description is required",
		},
		{
			name:        "Title exceeds limit",
			title:       strings.Repeat("a", MaxTitleLenght+1),
			description: "Test Description",
			expectError: true,
			errorMsg:    "title exceeds maximum",
		},
		{
			name:        "Description exceeds limit",
			title:       "Test Todo",
			description: strings.Repeat("a", MaxDescriptionLenght+1),
			expectError: true,
			errorMsg:    "description exceeds maximum",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := &mockStorage{}
			model := NewFormModel(mockStore)
			model.fields[titleField] = tt.title
			model.fields[descriptionField] = tt.description

			err := model.submitForm()

			if err == nil {
				t.Errorf("Expected error but got none")
			} else if !strings.Contains(err.Error(), tt.errorMsg) {
				t.Errorf("Expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
			}
		})
	}
}

func TestFormModel_CharacterCountDisplay(t *testing.T) {
	mockStore := &mockStorage{}
	model := NewFormModel(mockStore)

	model.fields[titleField] = "Test Title"
	model.fields[descriptionField] = "Test Description"

	view := model.View()

	expectedTitleCount := "10/100"
	expectedDescCount := "16/500"

	if !strings.Contains(view, expectedTitleCount) {
		t.Errorf("Expected to see title character count '%s' in view", expectedTitleCount)
	}

	if !strings.Contains(view, expectedDescCount) {
		t.Errorf("Expected to see description character count '%s' in view", expectedDescCount)
	}
}
