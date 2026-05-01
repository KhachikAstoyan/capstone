package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/KhachikAstoyan/capstone/internal/api/auth/domain"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

type StatsRepository interface {
	GetUserStats(ctx context.Context, userID uuid.UUID) (*domain.UserStats, error)
	GetSolvedByDifficulty(ctx context.Context, userID uuid.UUID) (map[string]int, error)
	GetSolvedByTag(ctx context.Context, userID uuid.UUID) ([]domain.TagStat, error)
	GetRecentSubmissions(ctx context.Context, userID uuid.UUID, limit int) ([]domain.RecentSubmission, error)
	GetSubmissionStats(ctx context.Context, userID uuid.UUID) (*domain.SubmissionStats, error)
}

type statsRepository struct {
	db *sql.DB
}

func NewStatsRepository(db *sql.DB) StatsRepository {
	return &statsRepository{db: db}
}

func (r *statsRepository) GetUserStats(ctx context.Context, userID uuid.UUID) (*domain.UserStats, error) {
	total, err := r.getTotalSolved(ctx, userID)
	if err != nil {
		return nil, err
	}

	byDifficulty, err := r.GetSolvedByDifficulty(ctx, userID)
	if err != nil {
		return nil, err
	}

	byTag, err := r.GetSolvedByTag(ctx, userID)
	if err != nil {
		return nil, err
	}

	recent, err := r.GetRecentSubmissions(ctx, userID, 10)
	if err != nil {
		return nil, err
	}

	subStats, err := r.GetSubmissionStats(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &domain.UserStats{
		TotalSolved:        total,
		SolvedByDifficulty: byDifficulty,
		SolvedByTag:        byTag,
		RecentSubmissions:  recent,
		SubmissionStats:    *subStats,
	}, nil
}

func (r *statsRepository) getTotalSolved(ctx context.Context, userID uuid.UUID) (int, error) {
	query := `
		SELECT COUNT(DISTINCT s.problem_id)
		FROM submissions s
		WHERE s.user_id = $1
		  AND s.kind = 'submit'
		  AND s.status = 'accepted'
	`
	var count int
	err := r.db.QueryRowContext(ctx, query, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get total solved: %w", err)
	}
	return count, nil
}

func (r *statsRepository) GetSolvedByDifficulty(ctx context.Context, userID uuid.UUID) (map[string]int, error) {
	query := `
		SELECT p.difficulty::text, COUNT(DISTINCT s.problem_id)
		FROM submissions s
		JOIN problems p ON p.id = s.problem_id
		WHERE s.user_id = $1
		  AND s.kind = 'submit'
		  AND s.status = 'accepted'
		  AND p.visibility = 'published'
		GROUP BY p.difficulty
	`
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get solved by difficulty: %w", err)
	}
	defer rows.Close()

	result := make(map[string]int)
	result["easy"] = 0
	result["medium"] = 0
	result["hard"] = 0

	for rows.Next() {
		var difficulty string
		var count int
		if err := rows.Scan(&difficulty, &count); err != nil {
			return nil, fmt.Errorf("failed to scan difficulty: %w", err)
		}
		result[difficulty] = count
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating difficulties: %w", err)
	}
	return result, nil
}

func (r *statsRepository) GetSolvedByTag(ctx context.Context, userID uuid.UUID) ([]domain.TagStat, error) {
	query := `
		SELECT t.id, t.name, COUNT(DISTINCT s.problem_id) as cnt,
		       ARRAY_AGG(DISTINCT s.problem_id ORDER BY s.problem_id) as problem_ids
		FROM submissions s
		JOIN problems p ON p.id = s.problem_id
		JOIN problem_tags pt ON pt.problem_id = p.id
		JOIN tags t ON t.id = pt.tag_id
		WHERE s.user_id = $1
		  AND s.kind = 'submit'
		  AND s.status = 'accepted'
		  AND p.visibility = 'published'
		GROUP BY t.id, t.name
		ORDER BY cnt DESC
	`
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get solved by tag: %w", err)
	}
	defer rows.Close()

	var stats []domain.TagStat
	for rows.Next() {
		var tagID uuid.UUID
		var tagName string
		var count int
		var problemIDs []uuid.UUID
		if err := rows.Scan(&tagID, &tagName, &count, pq.Array(&problemIDs)); err != nil {
			return nil, fmt.Errorf("failed to scan tag stat: %w", err)
		}
		stats = append(stats, domain.TagStat{
			TagID:    tagID,
			TagName:  tagName,
			Count:    count,
			Problems: nil,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating tags: %w", err)
	}
	return stats, nil
}

func (r *statsRepository) GetRecentSubmissions(ctx context.Context, userID uuid.UUID, limit int) ([]domain.RecentSubmission, error) {
	query := `
		SELECT DISTINCT ON (s.problem_id)
		       p.id, p.slug, p.title, l.key, s.status::text, s.created_at
		FROM submissions s
		JOIN problems p ON p.id = s.problem_id
		JOIN languages l ON l.id = s.language_id
		WHERE s.user_id = $1
		  AND s.kind = 'submit'
		  AND p.visibility = 'published'
		ORDER BY s.problem_id, s.created_at DESC
		LIMIT $2
	`
	rows, err := r.db.QueryContext(ctx, query, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent submissions: %w", err)
	}
	defer rows.Close()

	var submissions []domain.RecentSubmission
	for rows.Next() {
		var sub domain.RecentSubmission
		if err := rows.Scan(
			&sub.ProblemID,
			&sub.ProblemSlug,
			&sub.ProblemTitle,
			&sub.LanguageKey,
			&sub.Status,
			&sub.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan submission: %w", err)
		}
		submissions = append(submissions, sub)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating submissions: %w", err)
	}
	return submissions, nil
}

func (r *statsRepository) GetSubmissionStats(ctx context.Context, userID uuid.UUID) (*domain.SubmissionStats, error) {
	// Total submissions (submits)
	var totalSubmissions int
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM submissions
		WHERE user_id = $1 AND kind = 'submit'
	`, userID).Scan(&totalSubmissions)
	if err != nil {
		return nil, fmt.Errorf("failed to get total submissions: %w", err)
	}

	// Total test runs
	var totalTestRuns int
	err = r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM submissions
		WHERE user_id = $1 AND kind = 'run'
	`, userID).Scan(&totalTestRuns)
	if err != nil {
		return nil, fmt.Errorf("failed to get total test runs: %w", err)
	}

	// Acceptance rate
	var acceptedCount int
	err = r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM submissions
		WHERE user_id = $1 AND kind = 'submit' AND status = 'accepted'
	`, userID).Scan(&acceptedCount)
	if err != nil {
		return nil, fmt.Errorf("failed to get accepted count: %w", err)
	}

	acceptanceRate := 0.0
	if totalSubmissions > 0 {
		acceptanceRate = (float64(acceptedCount) / float64(totalSubmissions)) * 100
	}

	// Most used languages
	query := `
		SELECT l.key, l.display_name, COUNT(*) as cnt
		FROM submissions s
		JOIN languages l ON l.id = s.language_id
		WHERE s.user_id = $1
		GROUP BY l.key, l.display_name
		ORDER BY cnt DESC
		LIMIT 5
	`
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get most used languages: %w", err)
	}
	defer rows.Close()

	var languages []domain.LanguageStat
	for rows.Next() {
		var langKey, langName string
		var count int
		if err := rows.Scan(&langKey, &langName, &count); err != nil {
			return nil, fmt.Errorf("failed to scan language stat: %w", err)
		}
		languages = append(languages, domain.LanguageStat{
			LanguageKey:  langKey,
			LanguageName: langName,
			Count:        count,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating languages: %w", err)
	}

	return &domain.SubmissionStats{
		TotalSubmissions:  totalSubmissions,
		TotalTestRuns:     totalTestRuns,
		AcceptanceRate:    acceptanceRate,
		MostUsedLanguages: languages,
	}, nil
}
