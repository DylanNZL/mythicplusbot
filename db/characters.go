package db

import (
	"context"
	"fmt"
)

type Character struct {
	ID           int     `json:"id"`
	Name         string  `json:"name"`
	Realm        string  `json:"realm"`
	Class        string  `json:"class"`
	OverallScore float64 `json:"score"`
	TankScore    float64 `json:"tank_score"`
	DPSScore     float64 `json:"dps_score"`
	HealScore    float64 `json:"heal_score"`
	DateUpdated  int64   `json:"date_updated"`
	DateCreated  int64   `json:"date_created"`
}

const (
	getCharacterQuery = `SELECT id, name, realm, class, score, tank_score, dps_score, heal_score, date_updated, date_created FROM characters WHERE name=? AND realm=? LIMIT 1`

	updateCharacterQuery = `UPDATE characters SET score = ?, tank_score = ?, dps_score = ?, heal_score = ? WHERE name = ? AND realm = ?`

	deleteCharacterQuery = `DELETE FROM characters WHERE name = ? AND realm = ?`

	insertCharacterQuery = `INSERT INTO characters VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	listCharactersQuery = `SELECT id, name, realm, class, score, tank_score, dps_score, heal_score, date_updated, date_created FROM characters`
)

func (c *Character) IsEmpty() bool {
	return c.ID == 0 && c.Name == "" && c.Realm == "" && c.OverallScore == 0 && c.TankScore == 0 && c.DPSScore == 0 && c.HealScore == 0 && c.DateUpdated == 0 && c.DateCreated == 0
}

// CharacterRepo implements CharacterRepository interface
type CharacterRepo struct {
	db Database
}

// NewCharacterRepo creates a new character repository
func NewCharacterRepo(db Database) *CharacterRepo {
	return &CharacterRepo{db: db}
}

func (r *CharacterRepo) Insert(ctx context.Context, character *Character) error {
	return r.db.Query(ctx, insertCharacterQuery, character.ID, character.Name, character.Realm, character.Class,
		character.OverallScore, character.TankScore, character.DPSScore, character.HealScore, character.DateUpdated,
		character.DateCreated)
}

func (r *CharacterRepo) Update(ctx context.Context, character *Character) error {
	return r.db.Query(ctx, updateCharacterQuery, character.OverallScore, character.TankScore, character.DPSScore,
		character.HealScore, character.Name, character.Realm)
}

func (r *CharacterRepo) Delete(ctx context.Context, character *Character) error {
	return r.db.Query(ctx, deleteCharacterQuery,
		character.Name, character.Realm)
}

func (r *CharacterRepo) GetCharacter(ctx context.Context, name, realm string) (Character, error) {
	rows, err := r.db.QueryRows(ctx, getCharacterQuery, name, realm)
	if err != nil {
		return Character{}, err
	}
	defer rows.Close()

	if rows.Err() != nil {
		return Character{}, rows.Err()
	}

	if rows.Next() {
		var c Character
		if err := rows.Scan(&c.ID, &c.Name, &c.Realm, &c.Class, &c.OverallScore, &c.TankScore, &c.DPSScore, &c.HealScore,
			&c.DateUpdated, &c.DateCreated); err != nil {
			return c, err
		}
		return c, nil
	}

	return Character{}, nil // Character not found
}

func (r *CharacterRepo) CheckCharacterExists(ctx context.Context, name, realm string) (bool, error) {
	rows, err := r.db.QueryRows(ctx, "SELECT 1 FROM characters WHERE name=? AND realm=? LIMIT 1", name, realm)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	if rows.Err() != nil {
		return false, rows.Err()
	}

	return rows.Next(), nil
}

func (r *CharacterRepo) ListCharacters(ctx context.Context, limit int) ([]Character, error) {
	query := listCharactersQuery + " ORDER BY score DESC"

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := r.db.QueryRows(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var characters []Character
	for rows.Next() {
		var c Character
		if err := rows.Scan(&c.ID, &c.Name, &c.Realm, &c.Class, &c.OverallScore, &c.TankScore, &c.DPSScore, &c.HealScore,
			&c.DateUpdated, &c.DateCreated); err != nil {
			return nil, err
		}
		characters = append(characters, c)
	}

	return characters, rows.Err()
}
