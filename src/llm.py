import json
import logging
from pathlib import Path
from typing import Any
import cleaner
from openai import OpenAI
import os
import traceback
import dotenv

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

dotenv.load_dotenv()
client = OpenAI(
            api_key=os.getenv('GEMINI_KEY'),
            base_url="https://generativelanguage.googleapis.com/v1beta/openai/"
        )

def parse_job_desc(html_file_path: Path) -> str :
    try:
        with Path('./src/config/1_parsing.txt').open('r') as f:
            parsing_rules = f.read()

        with open(html_file_path, "r", encoding="utf-8") as file:
            html_content = file.read()
        relevant_content = cleaner.clean_html(html_content)
        logger.info(f"Reduced HTML to RELEVANT CONTENT: {len(html_content)}->{len(relevant_content)}")

        prompt = (
            f"{parsing_rules}\n\n"
            "Parse the following job description HTML, maintaining maximum detail while ensuring clean, normalized data:\n\n"
            f"{relevant_content}"
        )

        try:
            response = client.chat.completions.create(
                model="gemini-2.0-flash-lite",
                n=1,
                messages=[
                    {"role": "system", "content": "You are a helpful assistant."},
                    {
                        "role": "user",
                        "content": prompt
                    }
                ]
            )
        except Exception as e:
            logger.error(f"API call failed with: {str(e)}")
            logger.error(f"Full exception details: {traceback.format_exc()}")
            raise

        if response.choices[0].message.content is None:
            logger.error("Error: No content received in the OpenAI API response")
            raise
        clean_response = cleaner.clean_llm_response(response.choices[0].message.content)
        logger.debug("CLEANER RESPONSE")
        logger.debug(response.choices[0].message.content)
        return clean_response

    except Exception as e:
        logger.error(f"Unexpected error occurred at get_llm_response: {e}")
        logger.error("Traceback details:\n" + traceback.format_exc())
        raise

def get_feedback(parsed_data: dict) -> dict :
    try:
        with Path('./src/config/2_feedback.txt').open('r') as f:
            feedback_prompt = f.read()
        with Path('./src/config/resume.txt').open('r') as f:
            resume = f.read()


        messages :Any = [
            {"role": "system", "content": feedback_prompt}, 
            {"role": "user", "content": json.dumps({"parsed_job_desc": parsed_data})},
            {"role": "user", "content": resume},
        ]

        response = client.chat.completions.create(
            model="gemini-2.0-flash-lite",
            n=1,
            messages=messages,
        )

        if response.choices[0].message.content is None:
            logger.error("Error: No content received in the OpenAI API response")
            raise
        clean_response = cleaner.clean_llm_response(response.choices[0].message.content)
        feedback_data = json.loads(clean_response)
        print("TAILORED_RESPONSE")
        print(feedback_data)
        return feedback_data

    except Exception as e:
        error_msg = f"Unexpected error occurred in report function: {e}"
        logger.error(error_msg)
        logger.error("Traceback details:\n" + traceback.format_exc())
        raise type(e)(f"{error_msg}\nOrigin: {traceback.format_exc()}") from e

def get_tailored(feedback_data: dict) -> dict :
    try:
        with Path('./src/config/3_tailor.txt').open('r') as f:
            tailor_prompt = f.read()
        with Path('./src/config/resume.txt').open('r') as f:
            resume = f.read()

        messages: Any = [
            {"role": "system", "content": tailor_prompt}, 
            {"role": "user", "content": json.dumps({"feedback": feedback_data})},
            {"role": "user", "content": resume},
        ]

        response = client.chat.completions.create(
            model="gemini-2.0-flash-lite",
            n=1,
            messages=messages,
        )

        if response.choices[0].message.content is None:
            logger.error("Error: No content received in the OpenAI API response")
            raise
        clean_response = cleaner.clean_llm_response(response.choices[0].message.content)
        tailored_data = json.loads(clean_response)
        print("TAILORED_RESPONSE")
        print(tailored_data)
        return tailored_data

    except Exception as e:
        error_msg = f"Unexpected error occurred in tailoring LLM: {e}"
        logger.error(error_msg)
        logger.error("Traceback details:\n" + traceback.format_exc())
        raise type(e)(f"{error_msg}\nOrigin: {traceback.format_exc()}") from e


