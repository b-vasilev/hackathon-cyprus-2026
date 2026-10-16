package generation

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/passport"
)

var (
	// ErrFlagged prevents publishing a version that still needs review.
	ErrFlagged = errors.New("resolve the review flag before publishing this version")
	// ErrReviewInput reports an empty or oversized review note.
	ErrReviewInput = errors.New("invalid review note")
)

const maxReviewNote = 4000

// ReviewActivity is one comment or workflow change on a passport version.
type ReviewActivity struct {
	ID        string `json:"id"`
	Kind      string `json:"kind" jsonschema:"One of comment, edit, flag, resolve, publish, unpublish."`
	ChapterID string `json:"chapter_id"`
	Body      string `json:"body"`
	Author    string `json:"author"`
	CreatedAt string `json:"created_at"`
}

// Review holds reviewer-only state. Learners never receive it.
type Review struct {
	RevisionID string           `json:"revision_id"`
	FlagReason string           `json:"flag_reason"`
	FlaggedBy  string           `json:"flagged_by"`
	FlaggedAt  string           `json:"flagged_at"`
	EditedBy   string           `json:"edited_by"`
	EditedAt   string           `json:"edited_at"`
	Activity   []ReviewActivity `json:"activity"`
}

// CommentInput leaves a note on the whole passport or on one chapter.
type CommentInput struct {
	ChapterID string `json:"chapter_id"`
	Body      string `json:"body"`
}

// FlagInput explains what still needs review.
type FlagInput struct {
	Reason string `json:"reason"`
}

func now() string { return time.Now().UTC().Format(time.RFC3339Nano) }

func reviewNote(text string) (string, error) {
	text = strings.TrimSpace(text)
	if text == "" || len(text) > maxReviewNote {
		return "", fmt.Errorf("%w: write between 1 and %d characters", ErrReviewInput, maxReviewNote)
	}
	return text, nil
}

func addActivity(ctx context.Context, tx *sql.Tx, revisionID, userID, kind, chapterID, body string) error {
	_, err := tx.ExecContext(ctx, "INSERT INTO passport_review_activity(id,revision_id,user_id,kind,chapter_id,body,created_at) VALUES(?,?,?,?,?,?,?)", rand.Text(), revisionID, userID, kind, chapterID, body, now())
	if err != nil {
		return fmt.Errorf("record review activity: %w", err)
	}
	return nil
}

// change runs one reviewer update and its activity entry in a transaction.
func (s *Service) change(ctx context.Context, id string, apply func(*sql.Tx) error) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin review change: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	var exists string
	if readErr := tx.QueryRowContext(ctx, "SELECT id FROM passport_revisions WHERE id=?", id).Scan(&exists); readErr != nil {
		return fmt.Errorf("read passport version: %w", readErr)
	}
	if applyErr := apply(tx); applyErr != nil {
		return applyErr
	}
	if commitErr := tx.Commit(); commitErr != nil {
		return fmt.Errorf("commit review change: %w", commitErr)
	}
	return nil
}

// Publish shows the current working copy of this version to learners.
func (s *Service) Publish(ctx context.Context, id, userID string) (Revision, error) {
	err := s.change(ctx, id, func(tx *sql.Tx) error {
		var productID, flaggedAt, path, hash, publishedHash string
		var published bool
		if err := tx.QueryRowContext(ctx, `SELECT r.product_id,r.flagged_at,r.content_path,r.content_hash,r.published_content_hash,COALESCE(p.published_revision_id=r.id,0)
 FROM passport_revisions r JOIN products p ON p.id=r.product_id WHERE r.id=?`, id).Scan(&productID, &flaggedAt, &path, &hash, &publishedHash, &published); err != nil {
			return fmt.Errorf("read publication version: %w", err)
		}
		if flaggedAt != "" {
			return ErrFlagged
		}
		// Keep the old quiz with the old published text until this transaction.
		// Publishing edited text hides practice created from different content.
		if _, err := tx.ExecContext(ctx, `UPDATE learning_packs SET published_content_json='',published_source_content_hash='',published_by='',published_at=''
 WHERE revision_id=? AND published_source_content_hash<>?`, id, hash); err != nil {
			return fmt.Errorf("hide outdated practice: %w", err)
		}
		// Publish ready practice in the same transaction, including quiz-only changes.
		if _, err := tx.ExecContext(ctx, `UPDATE learning_packs SET published_content_json=content_json,published_source_content_hash=source_content_hash,published_by=?,
 published_at=CASE WHEN published_content_json=content_json AND published_source_content_hash=source_content_hash AND published_at<>'' THEN published_at ELSE ? END
 WHERE revision_id=? AND status='ready' AND content_json<>'' AND source_content_hash=? AND source_content_hash<>''`, userID, now(), id, hash); err != nil {
			return fmt.Errorf("publish passport practice: %w", err)
		}
		if published && (publishedHash == "" || publishedHash == hash) {
			return nil
		}
		if _, err := tx.ExecContext(ctx, "UPDATE products SET published_revision_id=? WHERE id=?", id, productID); err != nil {
			return fmt.Errorf("publish passport: %w", err)
		}
		if _, err := tx.ExecContext(ctx, "UPDATE passport_revisions SET published_by=?,published_at=?,published_content_path=?,published_content_hash=? WHERE id=?", userID, now(), path, hash, id); err != nil {
			return fmt.Errorf("record publication: %w", err)
		}
		return addActivity(ctx, tx, id, userID, "publish", "", "")
	})
	if err != nil {
		return Revision{}, err
	}
	return s.Revision(ctx, id)
}

// Unpublish hides this version from learners. Its content stays available to reviewers.
func (s *Service) Unpublish(ctx context.Context, id, userID string) (Revision, error) {
	err := s.change(ctx, id, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, "UPDATE products SET published_revision_id=NULL WHERE published_revision_id=?", id)
		if err != nil {
			return fmt.Errorf("unpublish passport: %w", err)
		}
		changed, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("read review change: %w", err)
		}
		if changed == 0 {
			return nil
		}
		if _, err := tx.ExecContext(ctx, "UPDATE passport_revisions SET published_content_path='',published_content_hash='' WHERE id=?", id); err != nil {
			return fmt.Errorf("clear published copy: %w", err)
		}
		return addActivity(ctx, tx, id, userID, "unpublish", "", "")
	})
	if err != nil {
		return Revision{}, err
	}
	return s.Revision(ctx, id)
}

// EditChapter saves reviewer text as a new immutable working-copy file.
func (s *Service) EditChapter(ctx context.Context, id, chapterID, userID string, edit passport.ChapterEdit) (Revision, error) {
	err := s.change(ctx, id, func(tx *sql.Tx) error {
		var productID, path, hash string
		var version int
		if err := tx.QueryRowContext(ctx, "SELECT product_id,version,content_path,content_hash FROM passport_revisions WHERE id=?", id).Scan(&productID, &version, &path, &hash); err != nil {
			return fmt.Errorf("read edited version: %w", err)
		}
		var content passport.Passport
		if err := s.Content.ReadJSON(path, hash, &content); err != nil {
			return fmt.Errorf("read edited passport: %w", err)
		}
		if content.Views.Document == nil {
			return fmt.Errorf("%w: this version uses an earlier structure; generate a new version to edit it", passport.ErrInvalidEdit)
		}
		if err := content.Views.Document.EditChapter(chapterID, edit); err != nil {
			return fmt.Errorf("edit chapter: %w", err)
		}
		editedPath := fmt.Sprintf("products/%s/versions/%d/edits/%s/passport.json", productID, version, rand.Text())
		editedHash, err := s.Content.WriteJSON(editedPath, content)
		if err != nil {
			return fmt.Errorf("save edited passport: %w", err)
		}
		// Older published versions have no captured copy yet. Capture it first so
		// learners keep the published text until this edit is published.
		if _, err := tx.ExecContext(ctx, "UPDATE passport_revisions SET published_content_path=content_path,published_content_hash=content_hash WHERE id=? AND published_content_hash='' AND id IN (SELECT published_revision_id FROM products WHERE published_revision_id IS NOT NULL)", id); err != nil {
			return fmt.Errorf("capture published copy: %w", err)
		}
		if _, err := tx.ExecContext(ctx, "UPDATE passport_revisions SET content_path=?,content_hash=?,edited_by=?,edited_at=? WHERE id=?", editedPath, editedHash, userID, now(), id); err != nil {
			return fmt.Errorf("save passport edit: %w", err)
		}
		return addActivity(ctx, tx, id, userID, "edit", chapterID, "")
	})
	if err != nil {
		return Revision{}, err
	}
	return s.Revision(ctx, id)
}

// Flag marks a version as needing review; it cannot be published until resolved.
func (s *Service) Flag(ctx context.Context, id, userID string, input FlagInput) (Review, error) {
	reason, err := reviewNote(input.Reason)
	if err != nil {
		return Review{}, err
	}
	err = s.change(ctx, id, func(tx *sql.Tx) error {
		if _, flagErr := tx.ExecContext(ctx, "UPDATE passport_revisions SET flag_reason=?,flagged_by=?,flagged_at=? WHERE id=?", reason, userID, now(), id); flagErr != nil {
			return fmt.Errorf("flag passport: %w", flagErr)
		}
		return addActivity(ctx, tx, id, userID, "flag", "", reason)
	})
	if err != nil {
		return Review{}, err
	}
	return s.Review(ctx, id)
}

// ResolveFlag clears the review flag.
func (s *Service) ResolveFlag(ctx context.Context, id, userID string) (Review, error) {
	err := s.change(ctx, id, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, "UPDATE passport_revisions SET flag_reason='',flagged_by='',flagged_at='' WHERE id=? AND flagged_at<>''", id)
		if err != nil {
			return fmt.Errorf("resolve passport flag: %w", err)
		}
		changed, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("read review change: %w", err)
		}
		if changed == 0 {
			return nil
		}
		return addActivity(ctx, tx, id, userID, "resolve", "", "")
	})
	if err != nil {
		return Review{}, err
	}
	return s.Review(ctx, id)
}

// Comment adds a reviewer note to the version.
func (s *Service) Comment(ctx context.Context, id, userID string, input CommentInput) (Review, error) {
	body, err := reviewNote(input.Body)
	if err != nil {
		return Review{}, err
	}
	if len(input.ChapterID) > 64 {
		return Review{}, fmt.Errorf("%w: unknown chapter", ErrReviewInput)
	}
	err = s.change(ctx, id, func(tx *sql.Tx) error {
		return addActivity(ctx, tx, id, userID, "comment", strings.TrimSpace(input.ChapterID), body)
	})
	if err != nil {
		return Review{}, err
	}
	return s.Review(ctx, id)
}

// Review returns the flag, last edit and activity of one version, oldest first.
func (s *Service) Review(ctx context.Context, id string) (Review, error) {
	review := Review{RevisionID: id, Activity: make([]ReviewActivity, 0)}
	err := s.DB.QueryRowContext(ctx, `SELECT r.flag_reason,COALESCE(f.email,''),r.flagged_at,COALESCE(e.email,''),r.edited_at
 FROM passport_revisions r LEFT JOIN users f ON f.id=r.flagged_by LEFT JOIN users e ON e.id=r.edited_by WHERE r.id=?`, id).Scan(&review.FlagReason, &review.FlaggedBy, &review.FlaggedAt, &review.EditedBy, &review.EditedAt)
	if err != nil {
		return review, fmt.Errorf("read passport review: %w", err)
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT a.id,a.kind,a.chapter_id,a.body,COALESCE(u.email,''),a.created_at
 FROM passport_review_activity a LEFT JOIN users u ON u.id=a.user_id WHERE a.revision_id=? ORDER BY a.created_at,a.rowid`, id)
	if err != nil {
		return review, fmt.Errorf("list review activity: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var item ReviewActivity
		if err := rows.Scan(&item.ID, &item.Kind, &item.ChapterID, &item.Body, &item.Author, &item.CreatedAt); err != nil {
			return review, fmt.Errorf("read review activity: %w", err)
		}
		review.Activity = append(review.Activity, item)
	}
	if err := rows.Err(); err != nil {
		return review, fmt.Errorf("read review activity: %w", err)
	}
	return review, nil
}
