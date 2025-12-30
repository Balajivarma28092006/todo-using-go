package storage

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/Balajivarma28092006/todo-using-go/internal/models"
	bolt "go.etcd.io/bbolt"
)

var (
	todoBucket   = []byte("todos")
	streakBucket = []byte("streaks")
)

type Strorage interface {
	SaveTodo(todo *models.Todo) error
	GetTodo(id string) (*models.Todo, error)
	GetAllTodos() ([]*models.Todo, error)
	UpdateTodo(todo *models.Todo) error
	DeleteTodo(id string) error
	GetStreak() (*Streak, error)
	UpdateStreak(streak *Streak) error
	Close() error
}

type BoltStorage struct {
	db *bolt.DB
}

type Streak struct {
	CurrentStreak    int            `json:"current_streak"`
	MaxStreak        int            `json:"max_streak"`
	LastCompletedAt  time.Time      `json:"last_completed_at"`
	TotalCompleted   int            `json:"total_completed"`
	DailyCompletions map[string]int `json:"daily_completions"`
}

func NewBoltStorage(dbPath string) (*BoltStorage, error) {
	db, err := bolt.Open(dbPath, 0o600, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to open a database: %w", err)
	}

	err = db.Update(func(tx *bolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists(todoBucket); err != nil {
			return err
		}
		if _, err := tx.CreateBucketIfNotExists(streakBucket); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create buckets: %w", err)
	}
	return &BoltStorage{db: db}, nil
}

func (s *BoltStorage) SaveTodo(todo *models.Todo) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(todoBucket)

		todo.CreatedAt = time.Now()
		todo.UpdatedAt = time.Now()

		data, err := json.Marshal(todo)
		if err != nil {
			return err
		}

		return b.Put([]byte(todo.ID), data)
	})
}

func (s *BoltStorage) GetTodo(id string) (*models.Todo, error) {
	var todo *models.Todo

	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(todoBucket)
		data := b.Get([]byte(id))

		if data == nil {
			return fmt.Errorf("todo not found")
		}

		todo = &models.Todo{}
		return json.Unmarshal(data, todo)
	})

	return todo, err
}

func (s *BoltStorage) GetAllTodos() ([]*models.Todo, error) {
	var todos []*models.Todo

	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(todoBucket)

		return b.ForEach(func(k, v []byte) error {
			var todo models.Todo
			if err := json.Unmarshal(v, &todo); err != nil {
				return err
			}
			todos = append(todos, &todo)
			return nil
		})
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(todos, func(i, j int) bool {
		if todos[i].Completed != todos[j].Completed {
			return !todos[i].Completed
		}

		if !todos[i].Completed {
			if todos[i].Deadline != nil && todos[j].Deadline != nil {
				return todos[i].Deadline.Before(*todos[j].Deadline)
			}
			if todos[i].Deadline != nil {
				return true
			}
			if todos[j].Deadline != nil {
				return false
			}
		}
		return todos[i].CreatedAt.After(todos[j].CreatedAt)
	})
	return todos, nil
}

func (s *BoltStorage) UpdateTodo(todo *models.Todo) error {
	var wasCompleted bool
	existingTodo, _ := s.GetTodo(todo.ID)
	if existingTodo != nil {
		wasCompleted = existingTodo.Completed
	}

	err := s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(todoBucket)

		todo.UpdatedAt = time.Now()

		data, err := json.Marshal(todo)
		if err != nil {
			return err
		}

		return b.Put([]byte(todo.ID), data)
	})

	if err != nil && !wasCompleted && todo.Completed {
		_ = s.updateStreakOnCompletion()
	}
	return err
}

func (s *BoltStorage) DeleteTodo(id string) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(todoBucket)
		return b.Delete([]byte(id))
	})
}

func (s *BoltStorage) GetStreak() (*Streak, error) {
	var streak *Streak

	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(streakBucket)
		data := b.Get([]byte("current"))

		if data == nil {
			streak = &Streak{
				CurrentStreak:    0,
				MaxStreak:        0,
				TotalCompleted:   0,
				DailyCompletions: make(map[string]int),
			}
			return nil
		}
		streak = &Streak{}
		return json.Unmarshal(data, streak)
	})
	return streak, err
}

func (s *BoltStorage) UpdateStreak(streak *Streak) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(streakBucket)

		data, err := json.Marshal(streak)
		if err != nil {
			return err
		}

		return b.Put([]byte("current"), data)
	})
}

func (s *BoltStorage) updateStreakOnCompletion() error {
	streak, err := s.GetStreak()
	if err != nil {
		return err
	}

	now := time.Now()
	today := now.Format("2000-01-19")

	if streak.DailyCompletions == nil {
		streak.DailyCompletions = make(map[string]int)
	}
	streak.DailyCompletions[today]++
	streak.TotalCompleted++

	if !streak.LastCompletedAt.IsZero() {
		daysSinceLastCompletion := int(now.Sub(streak.LastCompletedAt).Hours() / 24)

		if daysSinceLastCompletion == 0 {
		} else if daysSinceLastCompletion == 1 {
			streak.CurrentStreak++
			if streak.CurrentStreak > streak.MaxStreak {
				streak.MaxStreak = streak.CurrentStreak
			}
		} else {
			streak.CurrentStreak = 1
		}
	} else {
		streak.CurrentStreak = 1
		if streak.MaxStreak == 0 {
			streak.MaxStreak = 1
		}
	}
	streak.LastCompletedAt = now
	return s.UpdateStreak(streak)
}

func (s *BoltStorage) Close() error {
	return s.db.Close()
}

func GetTopUpcomingTodos(todos []*models.Todo, limit int) []*models.Todo {
	var upcomingTodos []*models.Todo
	for _, todo := range todos {
		if !todo.Completed && todo.Deadline != nil {
			upcomingTodos = append(upcomingTodos, todo)
		}
	}

	sort.Slice(upcomingTodos, func(i, j int) bool {
		if upcomingTodos[i].Deadline == nil || upcomingTodos[j].Deadline == nil {
			return false
		}
		return upcomingTodos[i].Deadline.Before(*upcomingTodos[j].Deadline)
	})

	if len(upcomingTodos) > limit {
		return upcomingTodos[:limit]
	}
	return upcomingTodos
}

func GetTodosWithoutDeadlines(todos []*models.Todo) []*models.Todo {
	var NoDeadlineTodos []*models.Todo
	for _, todo := range todos {
		if !todo.Completed && todo.Deadline == nil {
			NoDeadlineTodos = append(NoDeadlineTodos, todo)
		}
	}
	return NoDeadlineTodos
}
