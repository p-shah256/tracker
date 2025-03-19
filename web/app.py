import streamlit as st
import requests
import json
import re
from typing import Dict, Any

st.set_page_config(
    page_title="Resume Tailor",
    page_icon="📝",
    layout="wide",
    initial_sidebar_state="collapsed",
)

st.markdown(
    """
<style>
    .highlight {
        background-color: #f0f8ff;
        padding: 5px;
        border-radius: 5px;
        font-weight: bold;
        color: #0066cc;
    }
    .skill-tag {
        display: inline-block;
        padding: 3px 8px;
        margin: 2px;
        border-radius: 10px;
        font-size: 0.8em;
    }
    .required {
        background-color: #cfe2ff;
        border: 1px solid #9ec5fe;
    }
    .nice-to-have {
        background-color: #d1e7dd;
        border: 1px solid #a3cfbb;
    }
    .score-high {
        color: white;
        background-color: #198754;
        padding: 2px 6px;
        border-radius: 10px;
    }
    .score-medium {
        color: white;
        background-color: #fd7e14;
        padding: 2px 6px;
        border-radius: 10px;
    }
    .score-low {
        color: white;
        background-color: #dc3545;
        padding: 2px 6px;
        border-radius: 10px;
    }
</style>
""",
    unsafe_allow_html=True,
)

st.title("Resume Tailor 📝")
st.markdown("**Make your resume scream 'I'm perfect for this job!' without lying.**")

if "extracted_skills" not in st.session_state:
    st.session_state.extracted_skills = None
if "scored_resume" not in st.session_state:
    st.session_state.scored_resume = None
if "transformed_resume" not in st.session_state:
    st.session_state.transformed_resume = None

BACKEND_URL = "http://localhost:8080"


def call_api(endpoint: str, data: Dict[str, Any], is_form: bool = False) -> Dict[str, Any]:
    try:
        url = f"{BACKEND_URL}/api/{endpoint}"
        print(f"calling URL {url}")
        with st.spinner(f"Processing... This might take a few seconds"):
            if is_form:
                response = requests.post(url, data=data)
            else:
                response = requests.post(url, json=data)
            if response.status_code != 200:
                st.error(f"API Error: {response.status_code} - {response.text}")
                return None

            return response.json()
    except Exception as e:
        st.error(f"Error calling API: {str(e)}")
        return None


def format_score(score: int) -> str:
    if score >= 8:
        return f"<span class='score-high'>{score}</span>"
    elif score >= 5:
        return f"<span class='score-medium'>{score}</span>"
    else:
        return f"<span class='score-low'>{score}</span>"


def highlight_skills(text: str, skills: list) -> str:
    if not skills:
        return text

    result = text
    for skill in skills:
        pattern = re.compile(re.escape(skill), re.IGNORECASE)
        result = pattern.sub(f"<span class='highlight'>{skill}</span>", result)

    return result


tab1, tab2, tab3 = st.tabs(["Job Analysis", "Resume Matching", "Resume Transformation"])

with tab1:
    st.header("Step 1: Analyze Job Description")

    job_desc = st.text_area(
        "Paste the job description here",
        height=300,
        placeholder="Copy and paste the full job description from the job posting...",
        key="job_desc",
    )

    extract_btn = st.button("Extract Skills", type="primary")

    if extract_btn:
        if not job_desc.strip():
            st.error("Please enter a job description")
        else:
            extracted_data = call_api("extract", {"jobDescText": job_desc}, is_form=True)

            if extracted_data:
                st.session_state.extracted_skills = extracted_data
                st.success("Skills extracted successfully!")

    if st.session_state.extracted_skills:
        st.subheader("Extracted Skills")

        skills = st.session_state.extracted_skills
        company_info = skills.get("company_info", {})

        col1, col2, col3 = st.columns(3)
        with col1:
            st.info(f"**Company:** {company_info.get('name', 'N/A')}")
        with col2:
            st.info(f"**Position:** {company_info.get('position', 'N/A')}")
        with col3:
            st.info(f"**Level:** {company_info.get('level', 'N/A')}")

        # Required skills
        st.markdown("### Required Skills")
        required_skills_html = ""
        for skill in skills.get("required_skills", []):
            required_skills_html += (
                f"<span class='skill-tag required'>{skill['name']}</span> "
            )
        st.markdown(required_skills_html, unsafe_allow_html=True)

        # Nice-to-have skills
        st.markdown("### Nice-to-Have Skills")
        nice_skills_html = ""
        for skill in skills.get("nice_to_have_skills", []):
            nice_skills_html += (
                f"<span class='skill-tag nice-to-have'>{skill['name']}</span> "
            )
        st.markdown(nice_skills_html, unsafe_allow_html=True)
