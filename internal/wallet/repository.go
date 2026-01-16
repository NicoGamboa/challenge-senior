package wallet

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"os"
	"path/filepath"
	"sync"

	"challenge/kit/db"
)

type Repository interface {
	GetBalance(ctx context.Context, userID string) (int64, error)
	SetBalance(ctx context.Context, userID string, amount int64) error
	DebitIfSufficientFunds(ctx context.Context, userID string, amount int64) error
}

type SQLRepository struct {
	db db.Client
}

func NewSQLRepository(dbClient db.Client) *SQLRepository {
	return &SQLRepository{db: dbClient}
}

const (
	qWalletGetBalance = "SELECT balance FROM wallets WHERE user_id = ?"
	qWalletUpsert     = "INSERT INTO wallets (user_id, balance) VALUES (?, ?) ON DUPLICATE KEY UPDATE balance = ?"
	qWalletDebit      = "UPDATE wallets SET balance = balance - ? WHERE user_id = ? AND balance >= ?"
)

func (r *SQLRepository) GetBalance(ctx context.Context, userID string) (int64, error) {
	row, err := r.db.QueryRow(ctx, qWalletGetBalance, userID)
	if err != nil {
		log.Printf("layer=repo component=wallet repo=SQLRepository method=GetBalance user_id=%s err=%v", userID, err)
		return 0, err
	}
	var bal int64
	if err := row.Scan(&bal); err != nil {
		if db.IsNotFound(err) {
			return 0, nil
		}
		log.Printf("layer=repo component=wallet repo=SQLRepository method=GetBalance user_id=%s err=%v", userID, err)
		return 0, err
	}
	return bal, nil
}

func (r *SQLRepository) SetBalance(ctx context.Context, userID string, amount int64) error {
	if err := r.db.Exec(ctx, qWalletUpsert, userID, amount, amount); err != nil {
		log.Printf("layer=repo component=wallet repo=SQLRepository method=SetBalance user_id=%s amount=%d err=%v", userID, amount, err)
		return err
	}
	return nil
}

func (r *SQLRepository) DebitIfSufficientFunds(ctx context.Context, userID string, amount int64) error {
	if err := r.db.Exec(ctx, qWalletDebit, amount, userID, amount); err != nil {
		if db.IsConflict(err) {
			return ErrInsufficientFunds
		}
		log.Printf("layer=repo component=wallet repo=SQLRepository method=DebitIfSufficientFunds user_id=%s amount=%d err=%v", userID, amount, err)
		return err
	}
	return nil
}

type InMemoryRepository struct {
	mu       sync.Mutex
	balances map[string]int64
}

func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{balances: make(map[string]int64)}
}

func (r *InMemoryRepository) GetBalance(ctx context.Context, userID string) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.balances[userID], nil
}

func (r *InMemoryRepository) SetBalance(ctx context.Context, userID string, amount int64) error {
	r.mu.Lock()
	r.balances[userID] = amount
	r.mu.Unlock()
	return nil
}

func (r *InMemoryRepository) DebitIfSufficientFunds(ctx context.Context, userID string, amount int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cur := r.balances[userID]
	if err := ValidateSufficientFunds(cur, amount); err != nil {
		return err
	}
	r.balances[userID] = cur - amount
	return nil
}

type FileRepository struct {
	mu       sync.Mutex
	path     string
	balances map[string]int64
}

func NewFileRepository(path string) (*FileRepository, error) {
	r := &FileRepository{path: path, balances: make(map[string]int64)}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		log.Printf("layer=repo component=wallet repo=FileRepository method=NewFileRepository path=%s err=%v", path, err)
		return nil, err
	}
	if err := r.load(); err != nil {
		log.Printf("layer=repo component=wallet repo=FileRepository method=NewFileRepository path=%s err=%v", path, err)
		return nil, err
	}
	return r, nil
}

func (r *FileRepository) GetBalance(ctx context.Context, userID string) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.balances[userID], nil
}

func (r *FileRepository) SetBalance(ctx context.Context, userID string, amount int64) error {
	r.mu.Lock()
	r.balances[userID] = amount
	err := r.persistLocked()
	r.mu.Unlock()
	if err != nil {
		log.Printf("layer=repo component=wallet repo=FileRepository method=SetBalance user_id=%s amount=%d err=%v", userID, amount, err)
	}
	return err
}

func (r *FileRepository) DebitIfSufficientFunds(ctx context.Context, userID string, amount int64) error {
	r.mu.Lock()
	cur := r.balances[userID]
	if err := ValidateSufficientFunds(cur, amount); err != nil {
		r.mu.Unlock()
		return err
	}
	r.balances[userID] = cur - amount
	err := r.persistLocked()
	r.mu.Unlock()
	if err != nil {
		log.Printf("layer=repo component=wallet repo=FileRepository method=DebitIfSufficientFunds user_id=%s amount=%d err=%v", userID, amount, err)
	}
	return err
}

func (r *FileRepository) load() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	b, err := os.ReadFile(r.path)
	if err != nil {
		if os.IsNotExist(err) {
			if err := r.persistLocked(); err != nil {
				log.Printf("layer=repo component=wallet repo=FileRepository method=load path=%s err=%v", r.path, err)
				return errors.Join(db.ErrInternal, err)
			}
			return nil
		}
		log.Printf("layer=repo component=wallet repo=FileRepository method=load path=%s err=%v", r.path, err)
		return errors.Join(db.ErrInternal, err)
	}
	if len(b) == 0 {
		return nil
	}
	if err := json.Unmarshal(b, &r.balances); err != nil {
		log.Printf("layer=repo component=wallet repo=FileRepository method=load path=%s err=%v", r.path, err)
		return errors.Join(db.ErrInternal, err)
	}
	return nil
}

func (r *FileRepository) persistLocked() error {
	b, err := json.MarshalIndent(r.balances, "", "  ")
	if err != nil {
		log.Printf("layer=repo component=wallet repo=FileRepository method=persistLocked path=%s err=%v", r.path, err)
		return errors.Join(db.ErrInternal, err)
	}

	tmp := r.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		log.Printf("layer=repo component=wallet repo=FileRepository method=persistLocked path=%s err=%v", r.path, err)
		return errors.Join(db.ErrInternal, err)
	}
	if err := os.Rename(tmp, r.path); err != nil {
		log.Printf("layer=repo component=wallet repo=FileRepository method=persistLocked path=%s err=%v", r.path, err)
		return errors.Join(db.ErrInternal, err)
	}
	return nil
}
