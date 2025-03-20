import streamlit as st
import requests
import json
from typing import Dict, Any
import os
from dotenv import load_dotenv

# Load environment variables
load_dotenv()

# Get backend URL from environment variable
BACKEND_URL = os.getenv("BACKEND_URL", "http://localhost:8080")

# Define users from environment variables
USERS = {
    os.getenv("ADMIN_USERNAME"): {
        "password": os.getenv("ADMIN_PASSWORD"),
        "role": os.getenv("ADMIN_ROLE")
    },
    os.getenv("USER1_USERNAME"): {
        "password": os.getenv("USER1_PASSWORD"),
        "role": os.getenv("USER1_ROLE")
    },
    os.getenv("USER2_USERNAME"): {
        "password": os.getenv("USER2_PASSWORD"),
        "role": os.getenv("USER2_ROLE")
    }
}

# Remove any None values from USERS dict
USERS = {k: v for k, v in USERS.items() if k and v["password"] and v["role"]}

def initialize_session_state():
    """Initialize session state variables."""
    if "authenticated" not in st.session_state:
        st.session_state["authenticated"] = False
    if "username" not in st.session_state:
        st.session_state["username"] = ""
    if "user_role" not in st.session_state:
        st.session_state["user_role"] = ""

def check_login(username: str, password: str) -> bool:
    """Check if login credentials are correct."""
    if username in USERS and password == USERS[username]["password"]:
        st.session_state["username"] = username
        st.session_state["user_role"] = USERS[username]["role"]
        st.session_state["authenticated"] = True
        return True
    return False

def display_company_info(company_info: Dict[str, str]) -> None:
    """Display company information."""
    st.write(f"**Company:** {company_info.get('name', 'N/A')}")
    st.write(f"**Position:** {company_info.get('position', 'N/A')}")
    st.write(f"**Level:** {company_info.get('level', 'N/A')}")

def display_skills(skills: list, title: str) -> None:
    """Display skills with their context."""
    st.write(f"{title}:")
    for skill in skills:
        st.write(f"- {skill['name']}")
        if skill.get("context"):
            st.write(f"  *Context: {skill['context']}*")

def display_experience_highlights(highlights: list) -> None:
    """Display experience highlights with scores and reasoning."""
    for highlight in highlights:
        st.write(f"- {highlight.get('text', '')} (Score: {highlight.get('score', 0):.1f})")
        if highlight.get("reasoning"):
            st.write(f"  *Reasoning: {highlight.get('reasoning')}*")

def display_experience_scores(experience_list: list) -> None:
    """Display experience scores with matching skills and highlights."""
    st.write("Experience Scores:")
    for exp in experience_list:
        with st.expander(f"{exp.get('company', '')} - {exp.get('position', '')}: {exp.get('score', 0):.1f}/10"):
            st.write("Matching Skills:")
            for skill in exp.get("matching_skills", []):
                st.write(f"- {skill}")
            st.write("Highlights:")
            display_experience_highlights(exp.get("highlights", []))

def display_project_scores(projects: list) -> None:
    """Display project scores with matching skills and highlights."""
    st.write("Project Scores:")
    if not projects:
        st.write("No projects found")
        return

    for proj in projects:
        title = proj.get("name", "Unnamed Project")
        with st.expander(f"- {title}: {proj.get('score', 0):.1f}/10"):
            st.write("Matching Skills:")
            for skill in proj.get("matching_skills", []):
                st.write(f"- {skill}")
            st.write("Highlights:")
            display_experience_highlights(proj.get("highlights", []))

def display_transformed_item(item: Dict[str, Any]) -> None:
    """Display a transformed item with original and optimized versions."""
    title = f"{item.get('company', '')} - {item.get('position', '')}"
    if not title.strip():
        title = item.get("name", "Project")

    score_diff = item.get("new_score", 0) - item.get("original_score", 0)
    score_text = f"{item.get('id', 'N/A')} ({item.get('original_score', 0):.1f} ↗️ +{score_diff:.1f} = {item.get('new_score', 0):.1f})"

    with st.expander(f"{title} {score_text}"):
        col1, col2 = st.columns(2)
        with col1:
            st.write("**Original:**")
            st.write(item.get("original_text", ""))
            if item.get("original_skills"):
                st.write("**Original Skills:**")
                for skill in item.get("original_skills", []):
                    st.write(f"- {skill}")
        with col2:
            st.write("**Optimized:**")
            st.write(item.get("transformed_text", ""))
            if item.get("added_skills"):
                st.write("**Added Skills:**")
                for skill in item.get("added_skills", []):
                    st.write(f"- {skill} ✨")

        if item.get("reasoning"):
            st.write("**Reasoning:**")
            st.write(item.get("reasoning", ""))
        if item.get("improvement_explanation"):
            st.write("**Improvement Explanation:**")
            st.write(item.get("improvement_explanation", ""))

        if item.get("char_count_original") or item.get("char_count_new"):
            st.write("**Character Counts:**")
            st.write(f"- Original: {item.get('char_count_original', 0)}")
            st.write(f"- New: {item.get('char_count_new', 0)}")

def display_optimization_results(result: Dict[str, Any]) -> None:
    """Display all optimization results."""
    st.subheader("Extracted Skills")
    extracted_skills = result.get("extractedSkills", {})
    company_info = extracted_skills.get("company_info", {})
    display_company_info(company_info)

    col1, col2 = st.columns(2)
    with col1:
        display_skills(extracted_skills.get("required_skills", []), "Required Skills:")
    with col2:
        display_skills(extracted_skills.get("nice_to_have_skills", []), "Nice to Have Skills:")

    st.subheader("Resume Scoring")
    scored_resume = result.get("scoredResume", {})
    st.metric("Overall Resume Score", f"{scored_resume.get('overall_score', 0):.1f}/10")

    display_experience_scores(scored_resume.get("professional_experience", []))
    display_project_scores(scored_resume.get("projects", []))

    st.subheader("Suggested Optimizations Bullet Points")
    transformed_items = result.get("transformItems", [])
    for item in transformed_items:
        display_transformed_item(item)

def handle_optimize_request(jd_text: str, resume_text: str) -> None:
    """Handle the optimization request."""
    try:
        response = requests.post(
            f"{BACKEND_URL}/api/optimize",
            json={"jobDescText": jd_text, "resume": resume_text}
        )
        response.raise_for_status()
        result = response.json()
        display_optimization_results(result)
    except requests.exceptions.RequestException as e:
        st.error(f"Error communicating with backend: {str(e)}")

def render_login_page() -> None:
    """Render the login page."""
    st.title("Login")
    
    username = st.text_input("Username")
    password = st.text_input("Password", type="password")
    
    if st.button("Login"):
        if check_login(username, password):
            st.success(f"Welcome {username}!")
            st.rerun()
        else:
            st.error("Invalid username or password")

def render_main_app() -> None:
    """Render the main application."""
    st.set_page_config(page_title="Resume Optimizer", layout="wide")
    
    with st.sidebar:
        st.write(f"Logged in as: {st.session_state['username']} ({st.session_state['user_role']})")
        if st.button("Logout"):
            st.session_state["authenticated"] = False
            st.session_state["username"] = ""
            st.session_state["user_role"] = ""
            st.rerun()

    st.title("Resume Optimizer")
    st.markdown("Make your resume look like it was made for this specific job!")

    col1, col2 = st.columns(2)
    with col1:
        st.subheader("Job Description")
        jd_text = st.text_area("Paste the job description here", height=400)

    with col2:
        st.subheader("Your Resume")
        resume_text = st.text_area("Paste your resume here", height=400)

    if st.button("Optimize Resume"):
        handle_optimize_request(jd_text, resume_text)

def main():
    """Main application entry point."""
    initialize_session_state()

    if not st.session_state["authenticated"]:
        render_login_page()
    else:
        render_main_app()

if __name__ == "__main__":
    main()

