import json
import logging
import psycopg2
from psycopg2 import sql

# Setup logging - because we're professionals who debug! 
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


dummy_data = """{
  "db_friendly": {
    "company": "KAYAK",
    "position": {
      "name": "Java Backend Developer",
      "level": "Mid"
    },
    "skills": [
      {
        "name": "Java",
        "type": "technical",
        "priority": 5,
        "isMustHave": true,
        "yearsRequired": null,
        "context": "strong proficiency in Java and in-depth knowledge of data structures, concurrency, and OOP patterns"
      },
      {
        "name": "Data Structures",
        "type": "technical",
        "priority": 4,
        "isMustHave": true,
        "yearsRequired": null,
        "context": "in-depth knowledge of data structures"
      },
      {
        "name": "Concurrency",
        "type": "technical",
        "priority": 4,
        "isMustHave": true,
        "yearsRequired": null,
        "context": "in-depth knowledge of concurrency"
      }
    ]
  }
}"""

def not_none(value):
    if value is not None:
        return value
    raise ValueError(f"Value of {value} cannot be None")

def add_job_to_db(job_data: dict, message_id: int, conn: psycopg2.extensions.connection):
    cursor = conn.cursor()
    try:
        logger.info("Adding job to database")

        db_friendly = job_data.get("db_friendly", {})
        if not isinstance(db_friendly, dict):
            raise TypeError("Missing db_friendly data structure")
        logger.info(f"Adding LLMRESPONSE to DB: {json.dumps(db_friendly)}")

        # Type validation in one clean sweep
        company_name = db_friendly.get("company", "").strip()
        cursor.execute(sql.SQL("""INSERT INTO companies (name) VALUES (%s) ON CONFLICT (name) DO NOTHING"""), (company_name,))
        cursor.execute( sql.SQL("SELECT id FROM companies WHERE name ILIKE %s"), (company_name,))
        company_id = cursor.fetchone()
        company_id = not_none(company_id)[0]
        logger.debug(f"Company ID retrieved/created: {company_id}")

        # Skills: Batch process with list comprehension magic
        position_data = db_friendly.get("position", {})
        cursor.execute("""
            INSERT INTO job_applications (id, company_id, position_name, position_level, raw_json, location, remote_status) 
            VALUES (%s, %s, %s, %s, %s, %s, %s)
        """, (
            message_id, company_id, position_data.get("name", ""), position_data.get("level"), json.dumps(job_data),
            job_data.get("full_details", {}).get("metadata", {}).get("location", {}).get("primary"),
            job_data.get("full_details", {}).get("metadata", {}).get("location", {}).get("remote"),
        ))

        seen_skills = set()  # Our duplicate detector! ♂️
        for skill in db_friendly.get("skills", []):
            skill_name = skill["name"].strip()
            if skill_name in seen_skills:
                continue
            seen_skills.add(skill_name)
            cursor.execute( sql.SQL(""" INSERT INTO skills (name, type) VALUES (%s, %s) ON CONFLICT (name) DO NOTHING """), 
                (skill["name"].strip(), skill["type"]))

            cursor.execute( sql.SQL("SELECT id FROM skills WHERE name ILIKE %s"), (skill["name"],))
            skill_id = cursor.fetchone()
            skill_id = not_none(skill_id)[0]

            cursor.execute( sql.SQL("""
                    INSERT INTO job_skills ( job_id, skill_id, priority, is_must_have, years_required, proficiency_level, context
                    ) VALUES (%s, %s, %s, %s, %s, %s, %s) """), 
                           (
                            message_id, skill_id, skill.get("priority", 3), skill.get("isMustHave", False),
                            skill.get("yearsRequired"), skill.get("proficiencyLevel", ""), skill.get("context", "")
                           )
                    )

        conn.commit()
        logger.info("Job successfully added to database", 
                   extra={"message_id": message_id, "company": company_name})

    except Exception as e:
        logger.error(f"💥 Error adding job: {str(e)}", extra={"message_id": message_id})
        conn.rollback()
    finally:
        cursor.close()

# TODO: this is redundant. merge it
def process_job_posting(str_llm_content: str, message_id: int, db_config: dict)-> dict | None: 
    conn = psycopg2.connect(**db_config)
    try:
        job_data = json.loads(str_llm_content)
        add_job_to_db(job_data, message_id, conn)
        logger.info("Job posting processed successfully")
        return job_data.get("db_friendly", {})
    except Exception as e:
        logger.error(f"Error processing job posting: {e}")
        logger.debug(f"Raw JSON content (first 200 chars): {str_llm_content[:200]}...")
        conn.rollback()
    finally:
        conn.close()

def if_processed(message_id: int, db_config: dict) -> bool:
    conn = psycopg2.connect(**db_config)
    try:
        cursor = conn.cursor()
        cursor.execute(
            sql.SQL("SELECT id FROM job_applications WHERE id = %s"),
            (message_id,)
        )
        result_id = cursor.fetchone()
        logger.debug(f"Company ID retrieved/created: {result_id}")
        if result_id is None:
            logger.info(f"Message ID {message_id} does not exist in the database.")
            return False

        company_id = result_id[0]
        logger.debug(f"Message ID {message_id} exists with Company ID: {company_id}")
        return True
    except Exception as e:
        logger.error(f"Unexpected error while checking message_id {message_id}: {e}")
        return False
    finally:
        conn.close()
