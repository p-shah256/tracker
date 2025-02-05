import json
import logging
import psycopg2
from psycopg2 import sql

# Setup logging - because we're professionals who debug! 
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


def not_none(value):
    if value is not None:
        return value
    raise ValueError(f"Value of {value} cannot be None")

def add_job_to_db(parsed_data: dict, feedback_data: dict, tailored_data: dict, message_id: int, db_config: dict) -> dict|None:
    conn = psycopg2.connect(**db_config)
    cursor = conn.cursor()

    db_friendly = parsed_data.get("db_friendly", {})
    try:
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
            INSERT INTO job_applications (id, company_id, position_name, position_level, raw_json, feedback, tailored_bullets, ats_score)
            VALUES (%s, %s, %s, %s, %s, %s, %s, %s)
        """, (
                       message_id, company_id,
                       position_data.get("name", ""), position_data.get("level", ""), 
                       json.dumps(parsed_data),
                       json.dumps(feedback_data),
                       json.dumps(tailored_data), 
                       feedback_data.get("initial_score", 0)
        ))

        seen_skills = set()  # Our duplicate detector!
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
        return db_friendly

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
