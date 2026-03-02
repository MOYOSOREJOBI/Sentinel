package audit

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type chainEntry struct {
	id       int
	actor    string
	action   string
	details  string
	prevHash string
	entry    string
}

func Append(ctx context.Context, db *pgxpool.Pool, actor, action, details string) error {
	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, int64(731103)); err != nil {
		return err
	}

	var prev string
	_ = tx.QueryRow(ctx, `SELECT COALESCE(entry_hash,'') FROM audit_log ORDER BY id DESC LIMIT 1`).Scan(&prev)
	entry := hashFor(prev, actor, action, details)
	if _, err := tx.Exec(ctx, `INSERT INTO audit_log(actor,action,details,prev_hash,entry_hash) VALUES($1,$2,$3,$4,$5)`, actor, action, details, prev, entry); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func Verify(ctx context.Context, db *pgxpool.Pool) (bool, string, error) {
	rows, err := db.Query(ctx, `SELECT id, actor, action, details, prev_hash, entry_hash FROM audit_log ORDER BY id ASC`)
	if err != nil {
		return false, "", err
	}
	defer rows.Close()
	entries := make([]chainEntry, 0)
	for rows.Next() {
		var e chainEntry
		if err := rows.Scan(&e.id, &e.actor, &e.action, &e.details, &e.prevHash, &e.entry); err != nil {
			return false, "", err
		}
		entries = append(entries, e)
	}
	ok, msg := verifyChain(entries)
	return ok, msg, nil
}

func hashFor(prev, actor, action, details string) string {
	h := sha256.Sum256([]byte(prev + "|" + actor + "|" + action + "|" + details))
	return hex.EncodeToString(h[:])
}

func verifyChain(entries []chainEntry) (bool, string) {
	prev := ""
	for _, e := range entries {
		if e.prevHash != prev {
			return false, fmt.Sprintf("chain break at id %d", e.id)
		}
		if hashFor(prev, e.actor, e.action, e.details) != e.entry {
			return false, fmt.Sprintf("hash mismatch at id %d", e.id)
		}
		prev = e.entry
	}
	return true, "ok"
}
