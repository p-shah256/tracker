import json
import logging
from pathlib import Path
from typing import Any, Dict
import cleaner
from openai import OpenAI
import os
import traceback
import dotenv

logging.basicConfig(
    level=logging.DEBUG,
    format="%(asctime)s - %(levelname)s - %(name)s - %(filename)s:%(lineno)d - %(message)s",
    handlers=[logging.StreamHandler()],  # Output to console
)
logger = logging.getLogger(__name__)

dotenv.load_dotenv()
client = OpenAI(
    api_key=os.getenv("GEMINI_KEY"),
    base_url="https://generativelanguage.googleapis.com/v1beta/openai/",
)


def parse_job_desc(html_file_path: Path) -> Dict[str, Any]:
    try:
        parsing_rules_path = Path("./src/config/1_parsing.txt")
        if not parsing_rules_path.exists():
            raise FileNotFoundError(
                f"Parsing rules file not found at {parsing_rules_path}"
            )

        with parsing_rules_path.open("r", encoding="utf-8") as f:
            parsing_rules = f.read()

        try:
            with open(html_file_path, "r", encoding="utf-8") as file:
                html_content = file.read()
        except UnicodeDecodeError:
            with open(html_file_path, "r", encoding="latin-1") as file:
                html_content = file.read()

        relevant_content = cleaner.clean_html(html_content)
        logger.info(
            f"Reduced HTML from {len(html_content):,} to {len(relevant_content):,} chars"
        )

        prompt = (
            f"{parsing_rules}\n\n"
            "Parse the following job description HTML, maintaining maximum detail while ensuring clean, normalized data:\n\n"
            f"{relevant_content}"
        )

        logger.debug(f"Sending prompt to Gemini (length: {len(prompt)})")
        logger.debug(f"Prompt preview: {prompt[:500]}...")

        try:
            response = client.chat.completions.create(
                model="gemini-2.0-flash-lite",
                n=1,
                messages=[
                    {"role": "system", "content": "You are a helpful assistant."},
                    {"role": "user", "content": prompt},
                ],
            )
        except Exception as e:
            logger.error(f"Gemini API call failed: {str(e)}")
            logger.error(f"Full exception details: {traceback.format_exc()}")
            raise

        content = response.choices[0].message.content
        if content is None:
            logger.error("Error: Empty response from Gemini API")
            raise ValueError("Empty response from Gemini API")

        clean_response = cleaner.clean_llm_response(content)
        try:
            parsed_data = json.loads(clean_response)
            logger.debug(f"response recieved OK, cleaned JSON\n {parsed_data}")
            return parsed_data
        except json.JSONDecodeError as e:
            logger.error(f"Failed to parse LLM response as JSON: {e}")
            logger.debug(f"Invalid JSON: {clean_response[:500]}...")
            raise

    except Exception as e:
        logger.error(f"Job parsing failed: {e}")
        logger.error("Traceback details:\n" + traceback.format_exc())
        raise


def get_tailored(db_friendly: Dict[str, Any], relevant_yaml: Dict[str, Any]) -> str:
    try:
        tailor_path = Path("./src/config/3_tailor.txt")
        if not tailor_path.exists():
            raise FileNotFoundError(f"Tailor prompt file not found at {tailor_path}")

        with tailor_path.open("r", encoding="utf-8") as f:
            tailor_prompt = f.read().strip()

        db_friendly_str = json.dumps(db_friendly)
        relevant_yaml_str = json.dumps(relevant_yaml)

        logger.debug(f"Job data size: {len(db_friendly_str)} chars")
        logger.debug(f"Resume data size: {len(relevant_yaml_str)} chars")

        system_message = tailor_prompt

        user_message = (
            "I need to tailor a resume for a job. Here's the job data and resume segments:\n\n"
            f"JOB DATA: {db_friendly_str}\n\n"
            f"RESUME SECTIONS: {relevant_yaml_str}\n\n"
            "Please tailor the resume based on your tailoring instructions."
        )

        logger.debug(f"Sending tailoring request to Gemini ({len(user_message)} chars)")

        response = client.chat.completions.create(
            model="gemini-2.0-flash-lite",
            n=1,
            messages=[
                {"role": "system", "content": system_message},
                {"role": "user", "content": user_message},
            ],
        )

        content = response.choices[0].message.content
        if content is None:
            logger.error("Error: Empty tailoring response from Gemini API")
            raise ValueError("Empty tailoring response from Gemini API")

        try:
            return content
        except json.JSONDecodeError as e:
            logger.error(f"Failed to parse tailored response as JSON: {e}")
            logger.debug(f"Invalid JSON: {content[:500]}...")
            raise

    except Exception as e:
        error_msg = f"Resume tailoring failed: {e}"
        logger.error(error_msg)
        logger.error("Traceback details:\n" + traceback.format_exc())
        raise type(e)(f"{error_msg}\nOrigin: {traceback.format_exc()}") from e
