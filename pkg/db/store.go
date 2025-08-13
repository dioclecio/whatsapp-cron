package db

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"
)

const (
	dbFile = "data/mensagens.json"
)

type Mensagem struct {
	ID           int            `json:"id"`
	Destinatario string         `json:"destinatario"`
	Conteudos    []string       `json:"conteudos"`
	UltimoEnvio  string         `json:"ultimo_envio"`
	HorarioEnvio string         `json:"horario_envio"`
	DiaSemana    []time.Weekday `json:"dia_semana"`
}

type Database struct {
	Mensagens []Mensagem `json:"mensagens"`
}

type FileInfo struct {
	lastMod time.Time
}

func SecureFileAccess() error {
	// Ensure data directory exists
	dir := filepath.Dir(dbFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("error creating data directory: %v", err)
	}

	// Check if file exists
	_, err := os.Stat(dbFile)
	if os.IsNotExist(err) {
		// Create empty database
		db := Database{Mensagens: []Mensagem{}}
		if err := SaveDB(db); err != nil {
			return fmt.Errorf("error creating initial database: %v", err)
		}
	}

	// Set secure permissions
	if err := os.Chmod(dbFile, 0600); err != nil {
		return fmt.Errorf("error setting file permissions: %v", err)
	}

	return nil
}

func LoadDB() (Database, error) {
	if err := SecureFileAccess(); err != nil {
		return Database{}, fmt.Errorf("error securing file access: %v", err)
	}

	data, err := ioutil.ReadFile(dbFile)
	if err != nil {
		return Database{}, fmt.Errorf("error reading database file: %v", err)
	}

	var db Database
	if err := json.Unmarshal(data, &db); err != nil {
		return Database{}, fmt.Errorf("error parsing database JSON: %v", err)
	}

	return db, nil
}

func SaveDB(db Database) error {
	data, err := json.MarshalIndent(db, "", "  ")
	if err != nil {
		return fmt.Errorf("error encoding database JSON: %v", err)
	}

	if err := ioutil.WriteFile(dbFile, data, 0600); err != nil {
		return fmt.Errorf("error writing database file: %v", err)
	}

	return nil
}

func (fi *FileInfo) UpdateLastMod() error {
	info, err := os.Stat(dbFile)
	if err != nil {
		return fmt.Errorf("error getting file info: %v", err)
	}
	fi.lastMod = info.ModTime()
	return nil
}

func (fi *FileInfo) HasChanged() (bool, error) {
	info, err := os.Stat(dbFile)
	if err != nil {
		return false, fmt.Errorf("error getting file info: %v", err)
	}
	return info.ModTime().After(fi.lastMod), nil
}
