from fastapi import FastAPI, Form, UploadFile, File, HTTPException
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel
from typing import List, Dict, Any, Optional
import json
import os
import google.generativeai as genai
from dotenv import load_dotenv
import re

# Load environment variables
load_dotenv()

# Configure the API
app = FastAPI()

# Add CORS middleware to allow requests from Streamlit
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],  # Adjust this in production
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Configure Gemini
api_key = os.getenv("GEMINI_KEY")
if not api_key:
    raise ValueError("GEMINI_KEY environment variable not set")

genai.configure(api_key=api_key)
model = genai.GenerativeModel("gemini-2.0-flash")


# Helper class to clean LLM responses
class Cleaner:
    def clean_llm_response(self, response: str) -> str:
        # Detect if response is wrapped in code blocks
        if re.search(r"```(json)?\s*\{", response):
            # Extract JSON content from code blocks
            match = re.search(r"```(?:json)?\s*([\s\S]*?)```", response)
            if match:
                return match.group(1).strip()

        # Detect if response starts with a JSON object
        if response.strip().startswith("{"):
            return response.strip()

        return response


cleaner = Cleaner()


# Models
class ExtractedSkill(BaseModel):
    name: str
    importance: Optional[int] = None


class CompanyInfo(BaseModel):
    name: Optional[str] = None
    position: Optional[str] = None
    level: Optional[str] = None


class ExtractedSkills(BaseModel):
    required_skills: List[ExtractedSkill]
    nice_to_have_skills: List[ExtractedSkill]
    company_info: CompanyInfo


class Highlight(BaseModel):
    text: str
    score: int
    matching_skills: List[str]


class Experience(BaseModel):
    company: str
    position: str
    score: int
    matching_skills: List[str]
    highlights: List[Highlight]


class Project(BaseModel):
    name: str
    score: int
    matching_skills: List[str]
    highlights: List[Highlight]


class ScoredResume(BaseModel):
    professional_experience: List[Experience]
    projects: List[Project]


class TransformedHighlight(BaseModel):
    original: str
    transformed: str
    emphasized_skills: List[str]


class TransformedExperience(BaseModel):
    company: str
    position: str
    highlights: List[TransformedHighlight]


class TransformedProject(BaseModel):
    name: str
    highlights: List[TransformedHighlight]


class TransformedResume(BaseModel):
    professional_experience: List[TransformedExperience]
    projects: List[TransformedProject]


class MatchRequest(BaseModel):
    extracted_skills: ExtractedSkills
    resume_text: str


class TransformRequest(BaseModel):
    scored_resume: ScoredResume
    extracted_skills: ExtractedSkills
    min_score: int = 7


class AlternativeRequest(BaseModel):
    bullet_point: str
    matching_skills: List[str]


# API Endpoints
@app.post("/api/extract", response_model=ExtractedSkills)
async def extract_skills(jobDescText: str = Form(...)):
    """Extract skills from job description"""
    if not jobDescText:
        raise HTTPException(status_code=400, detail="No job description provided")

    prompt = f"""
    Extract skills and requirements from the following job description. 
    Classify them into "required_skills" and "nice_to_have_skills".
    Also extract company information like company name, position title, and level.
    
    Job Description:
    {jobDescText}
    
    Return the results as a JSON object with the following structure:
    {{
      "required_skills": [
        {{ "name": "skill name", "importance": 10 }}
      ],
      "nice_to_have_skills": [
        {{ "name": "skill name", "importance": 7 }}
      ],
      "company_info": {{
        "name": "company name",
        "position": "position title",
        "level": "job level"
      }}
    }}
    """

    try:
        response = model.generate_content(prompt)
        response_text = response.text

        # Clean the response
        cleaned_response = cleaner.clean_llm_response(response_text)

        # Parse JSON
        extracted_skills = json.loads(cleaned_response)

        return extracted_skills
    except Exception as e:
        raise HTTPException(
            status_code=500, detail=f"Failed to extract skills: {str(e)}"
        )


@app.post("/api/match", response_model=ScoredResume)
async def match_resume(extractedSkills: str = Form(...), resumeText: str = Form(...)):
    """Match resume against extracted skills"""
    if not extractedSkills:
        raise HTTPException(status_code=400, detail="No extracted skills provided")

    if not resumeText:
        raise HTTPException(status_code=400, detail="No resume text provided")

    try:
        # Parse extracted skills
        extracted_skills = json.loads(extractedSkills)

        prompt = f"""For each experience entry in my resume, identify which skills/requirements from the job description it addresses. 
Score each entry 1-10 on relevance, where 10 means it perfectly matches what the employer is looking for.

Job Requirements:
{json.dumps(extracted_skills)}

My Resume:
{resumeText}

Return the result as a JSON object with the following structure:
{{
  "professional_experience": [
    {{
      "company": "company name",
      "position": "position title",
      "score": 8,
      "matching_skills": ["skill1", "skill2"],
      "highlights": [
        {{
          "text": "original bullet point",
          "score": 7,
          "matching_skills": ["skill1"]
        }}
      ]
    }}
  ],
  "projects": [
    {{
      "name": "project name",
      "score": 9,
      "matching_skills": ["skill1", "skill3"],
      "highlights": [
        {{
          "text": "original bullet point",
          "score": 8,
          "matching_skills": ["skill1", "skill3"]
        }}
      ]
    }}
  ]
}}"""

        response = model.generate_content(
            [prompt], generation_config={"temperature": 0.2, "max_output_tokens": 8192}
        )
        response_text = response.text

        # Clean the response
        cleaned_response = cleaner.clean_llm_response(response_text)

        # Parse JSON
        scored_resume = json.loads(cleaned_response)

        return scored_resume
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Failed to match resume: {str(e)}")


@app.post("/api/transform", response_model=TransformedResume)
async def transform_resume(request: TransformRequest):
    """Transform high-scoring resume entries"""
    try:
        extracted_skills = request.extracted_skills
        scored_resume = request.scored_resume
        min_score = request.min_score

        prompt = f"""Transform these resume bullet points to better match the job requirements. 
For each bullet point with a score >= {min_score}, create an improved version that:
1. Emphasizes matching skills
2. Uses more impactful action verbs
3. Quantifies achievements where possible
4. Maintains factual accuracy

Job Requirements:
{json.dumps(extracted_skills)}

Resume Entries to Transform:
{json.dumps(scored_resume)}

Return the result as a JSON object with the following structure:
{{
  "professional_experience": [
    {{
      "company": "company name",
      "position": "position title",
      "highlights": [
        {{
          "original": "original bullet point",
          "transformed": "transformed bullet point",
          "emphasized_skills": ["skill1", "skill2"]
        }}
      ]
    }}
  ],
  "projects": [
    {{
      "name": "project name",
      "highlights": [
        {{
          "original": "original bullet point",
          "transformed": "transformed bullet point",
          "emphasized_skills": ["skill1", "skill3"]
        }}
      ]
    }}
  ]
}}"""

        response = model.generate_content(
            [prompt], generation_config={"temperature": 0.2, "max_output_tokens": 8192}
        )
        response_text = response.text

        # Clean the response
        cleaned_response = cleaner.clean_llm_response(response_text)

        # Parse JSON
        transformed_resume = json.loads(cleaned_response)

        return transformed_resume
    except Exception as e:
        raise HTTPException(
            status_code=500, detail=f"Failed to transform resume: {str(e)}"
        )


@app.post("/api/alternative")
async def generate_alternative(request: AlternativeRequest):
    """Generate alternative for a bullet point"""
    try:
        bullet_point = request.bullet_point
        matching_skills = request.matching_skills

        prompt = f"""Generate an alternative version of this resume bullet point that emphasizes the following skills: {', '.join(matching_skills)}.
Make it impactful, concise, and quantify achievements if possible.

Original bullet point:
{bullet_point}

Return just the alternative bullet point text with no extra formatting or explanation."""

        response = model.generate_content(
            [prompt], generation_config={"temperature": 0.7, "max_output_tokens": 1024}
        )
        response_text = response.text

        # Clean any quotes or formatting
        alternative = response_text.strip("\"'`").strip()

        return {"alternative": alternative}
    except Exception as e:
        raise HTTPException(
            status_code=500, detail=f"Failed to generate alternative: {str(e)}"
        )


# Run the app
if __name__ == "__main__":
    import uvicorn

    uvicorn.run(app, host="0.0.0.0", port=8080)
