import sqlite3
from pathlib import Path

def init_db_v2(db_path: str = "ats_killer_v2.db"):
    """Initializes the ATS Killer v2 database."""

    Path(db_path).parent.mkdir(parents=True, exist_ok=True)
    conn = sqlite3.connect(db_path)
    cursor = conn.cursor()

    cursor.execute("PRAGMA foreign_keys = ON;")

    tables = [
        """
        CREATE TABLE IF NOT EXISTS companies (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            name TEXT UNIQUE NOT NULL COLLATE NOCASE
        );
        """,
        """
        CREATE TABLE IF NOT EXISTS skills (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            name TEXT UNIQUE NOT NULL COLLATE NOCASE,
            type TEXT NOT NULL,
            CHECK (type IN ('technical', 'soft', 'domain'))
        );
        """,
        """
        CREATE TABLE IF NOT EXISTS job_applications (
            id BIGINT PRIMARY KEY,  -- Using message_id as our PK
            company_id INTEGER NOT NULL,
            position_name TEXT NOT NULL,
            position_level TEXT,
            raw_json JSON,  -- For all that juicy LLM response data
            location TEXT,
            remote_status BOOLEAN,
            compensation_min INTEGER,
            compensation_max INTEGER,
            compensation_currency TEXT,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            FOREIGN KEY (company_id) REFERENCES companies (id)
        );
        """,
        """
        CREATE TABLE IF NOT EXISTS job_skills (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            job_id BIGINT NOT NULL,
            skill_id INTEGER NOT NULL,
            priority INTEGER CHECK (priority BETWEEN 1 AND 5),
            is_must_have BOOLEAN DEFAULT 0,
            years_required INTEGER,
            proficiency_level TEXT,
            context TEXT,  -- Original requirement text, because context is king
            FOREIGN KEY (job_id) REFERENCES job_applications (id),
            FOREIGN KEY (skill_id) REFERENCES skills (id),
            UNIQUE(job_id, skill_id)
        );
        """
    ]

    for table in tables:
        cursor.execute(table)

    indices = [
        "CREATE INDEX IF NOT EXISTS idx_company_name ON companies(name);",
        "CREATE INDEX IF NOT EXISTS idx_skill_type ON skills(type);",
        "CREATE INDEX IF NOT EXISTS idx_job_skills ON job_skills(job_id, skill_id);",
        "CREATE INDEX IF NOT EXISTS idx_must_have_skills ON job_skills(is_must_have);",
        "CREATE INDEX IF NOT EXISTS idx_job_level ON job_applications(position_level);",
        "CREATE INDEX IF NOT EXISTS idx_job_company ON job_applications(company_id);"
    ]

    for index in indices:
        cursor.execute(index)

    conn.commit()
    conn.close()

    print("🚀 Database v2 initialized! Ready to wreak havoc on ATS systems!")


if __name__ == "__main__":
    init_db_v2()
