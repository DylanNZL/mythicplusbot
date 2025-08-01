package db

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestCharacterRepo_Insert(t *testing.T) {
	mockDB := &MockDatabase{}
	repo := NewCharacterRepo(mockDB)
	ctx := context.Background()

	character := &Character{
		ID:           1,
		Name:         "testchar",
		Realm:        "testrealm",
		Class:        "warrior",
		OverallScore: 2500.5,
		TankScore:    2400.0,
		DPSScore:     2300.0,
		HealScore:    0.0,
		DateUpdated:  1234567890,
		DateCreated:  1234567890,
	}

	mockDB.On("Query", ctx, "INSERT INTO characters VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		mock.MatchedBy(func(args []interface{}) bool {
			return len(args) == 10 &&
				args[0] == 1 &&
				args[1] == "testchar" &&
				args[2] == "testrealm" &&
				args[3] == "warrior" &&
				args[4] == 2500.5 &&
				args[5] == 2400.0 &&
				args[6] == 2300.0 &&
				args[7] == 0.0 &&
				args[8] == int64(1234567890) &&
				args[9] == int64(1234567890)
		})).Return(nil)

	err := repo.Insert(ctx, character)
	assert.NoError(t, err)
	mockDB.AssertExpectations(t)
}

func TestCharacterRepo_Update(t *testing.T) {
	mockDB := &MockDatabase{}
	repo := NewCharacterRepo(mockDB)
	ctx := context.Background()

	character := &Character{
		Name:         "testchar",
		Realm:        "testrealm",
		OverallScore: 2600.0,
		TankScore:    2500.0,
		DPSScore:     2400.0,
		HealScore:    0.0,
	}

	mockDB.On("Query", ctx, "UPDATE characters SET score = ?, tank_score = ?, dps_score = ?, heal_score = ? WHERE name = ? AND realm = ?",
		mock.MatchedBy(func(args []interface{}) bool {
			return len(args) == 6 &&
				args[0] == 2600.0 &&
				args[1] == 2500.0 &&
				args[2] == 2400.0 &&
				args[3] == 0.0 &&
				args[4] == "testchar" &&
				args[5] == "testrealm"
		})).Return(nil)

	err := repo.Update(ctx, character)
	assert.NoError(t, err)
	mockDB.AssertExpectations(t)
}

func TestCharacterRepo_Delete(t *testing.T) {
	mockDB := &MockDatabase{}
	repo := NewCharacterRepo(mockDB)
	ctx := context.Background()

	character := &Character{
		Name:  "testchar",
		Realm: "testrealm",
	}

	mockDB.On("Query", ctx, "DELETE FROM characters WHERE name = ? AND realm = ?",
		mock.MatchedBy(func(args []interface{}) bool {
			return len(args) == 2 &&
				args[0] == "testchar" &&
				args[1] == "testrealm"
		})).Return(nil)

	err := repo.Delete(ctx, character)
	assert.NoError(t, err)
	mockDB.AssertExpectations(t)
}

func TestCharacterRepo_GetCharacter_Found(t *testing.T) {
	mockDB := &MockDatabase{}
	repo := NewCharacterRepo(mockDB)
	ctx := context.Background()

	// Test the error case since mocking sql.Rows is complex
	mockDB.On("QueryRows", ctx, "SELECT id, name, realm, class, score, tank_score, dps_score, heal_score, date_updated, date_created FROM characters WHERE name=? AND realm=? LIMIT 1",
		mock.MatchedBy(func(args []interface{}) bool {
			return len(args) == 2 && args[0] == "testchar" && args[1] == "testrealm"
		})).Return((*sql.Rows)(nil), errors.New("mock error"))

	character, err := repo.GetCharacter(ctx, "testchar", "testrealm")
	assert.Error(t, err)
	assert.Equal(t, Character{}, character)
	mockDB.AssertExpectations(t)
}

func TestCharacterRepo_CheckCharacterExists(t *testing.T) {
	mockDB := &MockDatabase{}
	repo := NewCharacterRepo(mockDB)
	ctx := context.Background()

	mockDB.On("QueryRows", ctx, "SELECT 1 FROM characters WHERE name=? AND realm=? LIMIT 1",
		mock.MatchedBy(func(args []interface{}) bool {
			return len(args) == 2 && args[0] == "testchar" && args[1] == "testrealm"
		})).Return((*sql.Rows)(nil), errors.New("mock error"))

	exists, err := repo.CheckCharacterExists(ctx, "testchar", "testrealm")
	assert.Error(t, err)
	assert.False(t, exists)
	mockDB.AssertExpectations(t)
}

func TestCharacterRepo_ListCharacters_WithLimit(t *testing.T) {
	mockDB := &MockDatabase{}
	repo := NewCharacterRepo(mockDB)
	ctx := context.Background()

	expectedQuery := "SELECT id, name, realm, class, score, tank_score, dps_score, heal_score, date_updated, date_created FROM characters ORDER BY score DESC LIMIT 10"
	mockDB.On("QueryRows", ctx, expectedQuery, []interface{}(nil)).Return((*sql.Rows)(nil), errors.New("mock error"))

	characters, err := repo.ListCharacters(ctx, 10)
	assert.Error(t, err)
	assert.Nil(t, characters)
	mockDB.AssertExpectations(t)
}

func TestCharacterRepo_ListCharacters_NoLimit(t *testing.T) {
	mockDB := &MockDatabase{}
	repo := NewCharacterRepo(mockDB)
	ctx := context.Background()

	expectedQuery := "SELECT id, name, realm, class, score, tank_score, dps_score, heal_score, date_updated, date_created FROM characters ORDER BY score DESC"
	mockDB.On("QueryRows", ctx, expectedQuery, []interface{}(nil)).Return((*sql.Rows)(nil), errors.New("mock error"))

	characters, err := repo.ListCharacters(ctx, 0)
	assert.Error(t, err)
	assert.Nil(t, characters)
	mockDB.AssertExpectations(t)
}

// Test SQLiteDB implementation

func TestSQLiteDB_Query_NilDB(t *testing.T) {
	db := &SQLiteDB{db: nil}
	ctx := context.Background()

	err := db.Query(ctx, "SELECT 1")
	assert.ErrorIs(t, err, ErrNoDatabase)
}

func TestSQLiteDB_QueryRows_NilDB(t *testing.T) {
	db := &SQLiteDB{db: nil}
	ctx := context.Background()

	rows, err := db.QueryRows(ctx, "SELECT 1")
	assert.ErrorIs(t, err, ErrNoDatabase)
	assert.Nil(t, rows)
}

func TestSQLiteDB_Close_NilDB(t *testing.T) {
	db := &SQLiteDB{db: nil}
	err := db.Close()
	assert.NoError(t, err) // Should not error when db is nil
}

// Test Character struct methods

func TestCharacter_IsEmpty(t *testing.T) {
	tests := []struct {
		name      string
		character Character
		expected  bool
	}{
		{
			name:      "empty character",
			character: Character{},
			expected:  true,
		},
		{
			name: "character with ID only",
			character: Character{
				ID: 1,
			},
			expected: false,
		},
		{
			name: "character with name only",
			character: Character{
				Name: "testchar",
			},
			expected: false,
		},
		{
			name: "full character",
			character: Character{
				ID:           1,
				Name:         "testchar",
				Realm:        "testrealm",
				OverallScore: 2500.0,
				DateUpdated:  1234567890,
				DateCreated:  1234567890,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.character.IsEmpty()
			assert.Equal(t, tt.expected, result)
		})
	}
}
