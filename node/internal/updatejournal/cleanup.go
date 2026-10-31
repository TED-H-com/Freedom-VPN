package updatejournal

import (
	"context"
	"fmt"
)

func cleanupKey(jobID string, attempt int) string {
	return fmt.Sprintf("installer_cleanup/%s/%d", jobID, attempt)
}

// FinishInstallerJob seals the job and its pending-file cleanup obligation in
// one SQLite transaction. The filesystem rename is replayed after commit;
// SQLite and the filesystem cannot participate in one atomic transaction.
func (j *Journal) FinishInstallerJob(ctx context.Context, jobID string, attempt int, outcome, code string) error {
	tx, err := j.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	job, err := scanJob(tx.QueryRowContext(ctx, jobSelect+` WHERE job_id = ? AND attempt = ?`, jobID, attempt))
	if err != nil {
		return err
	}
	if job.Phase != JobDone {
		if _, err := tx.ExecContext(ctx, `UPDATE jobs SET phase = ?, outcome = ?, error_code = NULLIF(?, ''), updated_at = ? WHERE job_id = ? AND attempt = ?`,
			string(JobDone), outcome, code, j.ts(), jobID, attempt); err != nil {
			return err
		}
	}
	if err := setMeta(ctx, tx, cleanupKey(jobID, attempt), "1"); err != nil {
		return err
	}
	return tx.Commit()
}

// InstallerCleanups returns sealed jobs whose pending file needs consumption.
func (j *Journal) InstallerCleanups(ctx context.Context) ([]Job, error) {
	rows, err := j.db.QueryContext(ctx, jobSelect+` WHERE phase = ? AND EXISTS
		(SELECT 1 FROM meta WHERE key = 'installer_cleanup/' || jobs.job_id || '/' || jobs.attempt) ORDER BY rowid`, string(JobDone))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var jobs []Job
	for rows.Next() {
		job, err := scanJob(rows)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}
	return jobs, rows.Err()
}

// ClearInstallerCleanup is called only after the pending-file rename and
// directory fsync (or when that exact record has already disappeared).
func (j *Journal) ClearInstallerCleanup(ctx context.Context, jobID string, attempt int) error {
	_, err := j.db.ExecContext(ctx, `DELETE FROM meta WHERE key = ?`, cleanupKey(jobID, attempt))
	return err
}
