-- Keep at most one mother, father, and guardian per student (sponsor/other remain unrestricted).
DELETE FROM student_guardians sg
WHERE sg.id IN (
    SELECT id FROM (
        SELECT id,
               ROW_NUMBER() OVER (
                   PARTITION BY student_id, relationship
                   ORDER BY created_at ASC, id ASC
               ) AS rn
        FROM student_guardians
        WHERE relationship IN ('mother', 'father', 'guardian')
    ) d
    WHERE d.rn > 1
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_unique_student_exclusive_relationship
    ON student_guardians (student_id, relationship)
    WHERE relationship IN ('mother', 'father', 'guardian');
