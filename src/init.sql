SET CONSTRAINTS ALL IMMEDIATE;

CREATE TABLE IF NOT EXISTS companies (
    id SERIAL PRIMARY KEY,
    name TEXT UNIQUE NOT NULL
);

CREATE TABLE IF NOT EXISTS skills (
    id SERIAL PRIMARY KEY,
    name TEXT UNIQUE NOT NULL,
    type TEXT NOT NULL CHECK (type IN ('technical', 'soft', 'domain'))
);

CREATE TABLE IF NOT EXISTS job_applications (
    id BIGINT PRIMARY KEY,  -- Using message_id as our PK
    company_id INTEGER NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    position_name TEXT NOT NULL,
    position_level INTEGER,  -- Changed from TEXT to INT
    raw_json JSONB,  -- Using JSONB for better performance and indexing
    location TEXT,
    remote_status BOOLEAN,
    compensation_min INTEGER,
    compensation_max INTEGER,
    compensation_currency TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS job_skills (
    id SERIAL PRIMARY KEY,
    job_id BIGINT NOT NULL REFERENCES job_applications(id) ON DELETE CASCADE,
    skill_id INTEGER NOT NULL REFERENCES skills(id) ON DELETE CASCADE,
    priority INTEGER CHECK (priority BETWEEN 1 AND 5),
    is_must_have BOOLEAN DEFAULT FALSE,
    years_required INTEGER,
    proficiency_level TEXT,
    context TEXT,  -- Original requirement text
    UNIQUE(job_id, skill_id)
);

CREATE INDEX IF NOT EXISTS idx_company_name ON companies(name);
CREATE INDEX IF NOT EXISTS idx_skill_type ON skills(type);
CREATE INDEX IF NOT EXISTS idx_job_skills ON job_skills(job_id, skill_id);
CREATE INDEX IF NOT EXISTS idx_must_have_skills ON job_skills(is_must_have);
CREATE INDEX IF NOT EXISTS idx_job_level ON job_applications(position_level);
CREATE INDEX IF NOT EXISTS idx_job_company ON job_applications(company_id);

DO $$ BEGIN
    RAISE NOTICE '🚀 Database v2 initialized! Ready to wreak havoc on ATS systems!';
END $$;
