-- NULL/disabled archives stay unknown, never zero. Used by verify and the proof.
SELECT CASE WHEN current_setting('archive_mode') IN ('on', 'always')
                 AND last_archived_time IS NOT NULL AND last_archived_wal IS NOT NULL
            THEN (EXTRACT(EPOCH FROM (clock_timestamp() - last_archived_time)) * 1000)::bigint
       END
FROM pg_stat_archiver;
