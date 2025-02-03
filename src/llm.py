import json
import logging
from pathlib import Path
import cleaner
from openai import OpenAI
import os
import traceback
from string import Template

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

def parse(html_file_path: Path) -> str :
    try:
        json_struct = Path('./src/config/json_struct.txt')
        extraction_rules = Path('./src/config/extraction_rules.txt')
        with json_struct.open('r') as f:
            json_struct = Template(f.read())
        with extraction_rules.open('r') as f:
            extraction_rules = Template(f.read())

        client = OpenAI(
            api_key=os.getenv('GEMINI_KEY'),
            base_url="https://generativelanguage.googleapis.com/v1beta/openai/"
        )
        with open(html_file_path, "r", encoding="utf-8") as file:
            html_content = file.read()
        relevant_content = cleaner.clean_html(html_content)
        logger.info(f"Reduced HTML to RELEVANT CONTENT: {len(html_content)}->{len(relevant_content)}")
        prompt = f"""
            You are an elite ATS system reverse engineer with a PhD in Job Description Deconstruction™. Your mission is to parse this HTML into structured data that would make even the pickiest type system happy. Output ONLY valid JSON matching this structure (no explanations/text):
            {prompt_path}
            {extraction_rules}
            Parse the following job description HTML, maintaining maximum detail while ensuring clean, normalized data. Output ONLY valid JSON matching this structure:
            {relevant_content}
            """

        try:
            response = client.chat.completions.create(
                model="gemini-1.5-flash",
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
        return clean_response

    except Exception as e:
        logger.error(f"Unexpected error occurred at get_llm_response: {e}")
        logger.error("Traceback details:\n" + traceback.format_exc())
        raise

def report(db_freindly: dict) -> dict :
    try:
        prompt_path = Path('./src/config/tailor_prompt.txt')
        resume_path = Path('./src/config/resume.txt')
        with prompt_path.open('r') as f:
            template = Template(f.read())
        with resume_path.open('r') as f:
            resume = Template(f.read())

        prompt = template.substitute(
            JOB_DESCRIPTION=json.dumps(db_freindly),
            RESUME=resume
        )
        client = OpenAI(
            api_key=os.getenv('GEMINI_KEY'),
            base_url="https://generativelanguage.googleapis.com/v1beta/openai/"
        )

        response = client.chat.completions.create(
            model="gemini-1.5-flash",
            n=1,
            messages=[ {"role": "system", "content": "You are a helpful assistant."},
                { "role": "user", "content": prompt }
            ]
        )

        if response.choices[0].message.content is None:
            logger.error("Error: No content received in the OpenAI API response")
            raise
        clean_response = cleaner.clean_llm_response(response.choices[0].message.content)
        logger.info(clean_response)
        report_data = json.loads(clean_response)
        return report_data

    except Exception as e:
        logger.error(f"Unexpected error occurred at get_llm_response: {e}")
        logger.error("Traceback details:\n" + traceback.format_exc())
        raise
