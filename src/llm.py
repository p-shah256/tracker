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

json_structure = """
Expected Output Structure:
{
  "db_friendly": {
    "company": string,
    "position": {
      "name": string,
      "level": int  // years of experience.. see details for rule 0
    },
    "skills": [
      {
        "name": string,       // e.g., "JavaScript", "Problem Solving"
        "type": string,       // "technical", "soft", or "domain"
        "priority": number,   // 1-5, where 5 is "they'll sell their soul for this skill"
        "isMustHave": boolean,
        "yearsRequired": number | null,
        "context": string     // original requirement text, ATS loves this stuff
      }
    ]
  },
  "full_details": {
    "metadata": {
      "company": string,
      "role": string,
      "level": int,
      "location": {
        "primary": string,
        "remote": boolean,
        "travel": string
      },
      "posted": string,
      "industryContext": {
        "domain": [string],
        "specificTerms": [string],
        "operationalEnvironment": string
      }
    },
    "requirements": {
      "technicalSkills": [{
        "skill": string,
        "type": string,
        "context": string,
        "proficiencyLevel": string,
        "yearsRequired": number,
        "priority": number,
        "isMustHave": boolean
      }],
      "softSkills": [{
        "skill": string,
        "context": string,
        "priority": number,
        "isMustHave": boolean
      }]
    },
    "responsibilities": [{
      "description": string,
      "impliedSkills": [string],
      "domain": string,
      "priority": number,
      "keyTerms": [string]
    }],
    "compensation": {
      "ranges": [{
        "level": string,
        "min": number,
        "max": number,
        "currency": string
      }]
    }
  }
}
"""

extraction_rules = """
Extraction Rules:
0. Experience Calculation:
   - Look for explicit overall experience requirement (e.g. "5+ years experience required")
   - If not found, identify the highest years required from any must-have technical skill
   - If still not found, extract implied years from language like "senior" (8+), "mid" (4-7), "junior" (0-3)
1. Atomize compound skills: "React/Vue/Angular" → three separate entries
2. Detect proficiency levels from context:
   - "Expert in" → "Expert"
   - "Knowledge of" → "Familiar"
   - "Strong" → "Proficient"
3. Priority Scoring (1-5):
   - Position (earlier = higher)
   - Language strength ("must" > "should" > "nice")
   - Visual emphasis (bold, repetition)
   - Section importance (requirements > nice-to-have)
4. Implied Skills:
   - Extract skills from responsibility descriptions
   - Map domain-specific terminology
   - Identify underlying technical requirements
5. Context Preservation:
   - Always keep original phrasing in "context" field
   - Maintain relationship between requirements
"""

def parse(html_file_path: Path) -> str :
    try:
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
            {json_structure}
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

    except FileNotFoundError as e:
        logger.error(f"File not found: {html_file_path}. Error: {e}")
        raise 
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

        try:
            response = client.chat.completions.create(
                model="gemini-1.5-flash",
                n=1,
                messages=[ {"role": "system", "content": "You are a helpful assistant."},
                    { "role": "user", "content": prompt }
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
        logger.info(clean_response)
        report_data = json.loads(clean_response)
        return report_data

    except FileNotFoundError as e:
        logger.error(f"File not found: resume_path or prompt_path. Error: {e}")
        raise 
    except Exception as e:
        logger.error(f"Unexpected error occurred at get_llm_response: {e}")
        logger.error("Traceback details:\n" + traceback.format_exc())
        raise
